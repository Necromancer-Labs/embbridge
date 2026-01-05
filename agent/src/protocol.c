/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Protocol handling and minimal MessagePack implementation
 *
 * We implement a minimal subset of MessagePack to avoid external dependencies.
 * This keeps the binary small for embedded systems.
 *
 * Wire Protocol:
 *   All messages are length-prefixed: [4 bytes BE length][MessagePack payload]
 *
 * Message Types:
 *   - hello:     Agent -> Client handshake initiation
 *   - hello_ack: Client -> Agent handshake response
 *   - req:       Client -> Agent command request
 *   - resp:      Agent -> Client command response
 *   - data:      Chunked file transfer (both directions)
 *
 * This file provides:
 *   - MessagePack writer (mp_writer_t) for encoding responses
 *   - MessagePack reader (mp_reader_t) for decoding requests
 *   - Wire protocol send/receive with length framing
 *   - High-level message builders (proto_send_hello, proto_send_response, etc)
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <arpa/inet.h>

#include "edb.h"

/* =============================================================================
 * MessagePack Format Constants
 * ============================================================================= */

/* Format markers */
#define MP_FIXMAP       0x80
#define MP_FIXARRAY     0x90
#define MP_FIXSTR       0xa0
#define MP_NIL          0xc0
#define MP_FALSE        0xc2
#define MP_TRUE         0xc3
#define MP_BIN8         0xc4
#define MP_BIN16        0xc5
#define MP_BIN32        0xc6
#define MP_UINT8        0xcc
#define MP_UINT16       0xcd
#define MP_UINT32       0xce
#define MP_UINT64       0xcf
#define MP_INT8         0xd0
#define MP_INT16        0xd1
#define MP_INT32        0xd2
#define MP_INT64        0xd3
#define MP_STR8         0xd9
#define MP_STR16        0xda
#define MP_STR32        0xdb
#define MP_ARRAY16      0xdc
#define MP_ARRAY32      0xdd
#define MP_MAP16        0xde
#define MP_MAP32        0xdf

/* =============================================================================
 * MessagePack Writer
 *
 * A simple growable buffer for building MessagePack-encoded messages.
 * Used by proto_send_* functions to construct outgoing messages.
 *
 * Usage:
 *   mp_writer_t w;
 *   mp_writer_init(&w, 64);
 *   mp_write_map(&w, 2);
 *   mp_write_str(&w, "key1");
 *   mp_write_str(&w, "value1");
 *   mp_write_str(&w, "key2");
 *   mp_write_uint(&w, 42);
 *   proto_send(conn, w.buf, w.len);
 *   mp_writer_free(&w);
 * ============================================================================= */

typedef struct {
    uint8_t *buf;   /* Buffer holding encoded data */
    size_t   cap;   /* Allocated capacity */
    size_t   len;   /* Current length of data */
} mp_writer_t;

static int mp_writer_init(mp_writer_t *w, size_t initial_cap)
{
    w->buf = malloc(initial_cap);
    if (!w->buf) return -1;
    w->cap = initial_cap;
    w->len = 0;
    return 0;
}

static void mp_writer_free(mp_writer_t *w)
{
    if (w->buf) free(w->buf);
    w->buf = NULL;
    w->cap = 0;
    w->len = 0;
}

static int mp_writer_ensure(mp_writer_t *w, size_t need)
{
    if (w->len + need <= w->cap) return 0;

    size_t new_cap = w->cap * 2;
    while (new_cap < w->len + need) new_cap *= 2;

    uint8_t *new_buf = realloc(w->buf, new_cap);
    if (!new_buf) return -1;

    w->buf = new_buf;
    w->cap = new_cap;
    return 0;
}

static int mp_write_raw(mp_writer_t *w, const void *data, size_t len)
{
    if (mp_writer_ensure(w, len) < 0) return -1;
    memcpy(w->buf + w->len, data, len);
    w->len += len;
    return 0;
}

static int mp_write_u8(mp_writer_t *w, uint8_t val)
{
    return mp_write_raw(w, &val, 1);
}

static int mp_write_u16be(mp_writer_t *w, uint16_t val)
{
    uint16_t be = htons(val);
    return mp_write_raw(w, &be, 2);
}

static int mp_write_u32be(mp_writer_t *w, uint32_t val)
{
    uint32_t be = htonl(val);
    return mp_write_raw(w, &be, 4);
}

/* Write nil */
static int mp_write_nil(mp_writer_t *w)
{
    return mp_write_u8(w, MP_NIL);
}

/* Write boolean */
static int mp_write_bool(mp_writer_t *w, bool val)
{
    return mp_write_u8(w, val ? MP_TRUE : MP_FALSE);
}

/* Write unsigned integer */
static int mp_write_uint(mp_writer_t *w, uint64_t val)
{
    if (val <= 0x7f) {
        return mp_write_u8(w, (uint8_t)val);
    } else if (val <= 0xff) {
        if (mp_write_u8(w, MP_UINT8) < 0) return -1;
        return mp_write_u8(w, (uint8_t)val);
    } else if (val <= 0xffff) {
        if (mp_write_u8(w, MP_UINT16) < 0) return -1;
        return mp_write_u16be(w, (uint16_t)val);
    } else if (val <= 0xffffffff) {
        if (mp_write_u8(w, MP_UINT32) < 0) return -1;
        return mp_write_u32be(w, (uint32_t)val);
    } else {
        /* 64-bit, write as two 32-bit values */
        if (mp_write_u8(w, MP_UINT64) < 0) return -1;
        if (mp_write_u32be(w, (uint32_t)(val >> 32)) < 0) return -1;
        return mp_write_u32be(w, (uint32_t)val);
    }
}

/* Write string */
static int mp_write_str(mp_writer_t *w, const char *str)
{
    size_t len = strlen(str);

    if (len <= 31) {
        if (mp_write_u8(w, MP_FIXSTR | (uint8_t)len) < 0) return -1;
    } else if (len <= 0xff) {
        if (mp_write_u8(w, MP_STR8) < 0) return -1;
        if (mp_write_u8(w, (uint8_t)len) < 0) return -1;
    } else if (len <= 0xffff) {
        if (mp_write_u8(w, MP_STR16) < 0) return -1;
        if (mp_write_u16be(w, (uint16_t)len) < 0) return -1;
    } else {
        if (mp_write_u8(w, MP_STR32) < 0) return -1;
        if (mp_write_u32be(w, (uint32_t)len) < 0) return -1;
    }

    return mp_write_raw(w, str, len);
}

/* Write binary data */
static int mp_write_bin(mp_writer_t *w, const uint8_t *data, size_t len)
{
    if (len <= 0xff) {
        if (mp_write_u8(w, MP_BIN8) < 0) return -1;
        if (mp_write_u8(w, (uint8_t)len) < 0) return -1;
    } else if (len <= 0xffff) {
        if (mp_write_u8(w, MP_BIN16) < 0) return -1;
        if (mp_write_u16be(w, (uint16_t)len) < 0) return -1;
    } else {
        if (mp_write_u8(w, MP_BIN32) < 0) return -1;
        if (mp_write_u32be(w, (uint32_t)len) < 0) return -1;
    }

    return mp_write_raw(w, data, len);
}

/* Write map header (caller must write key-value pairs after) */
static int mp_write_map(mp_writer_t *w, size_t count)
{
    if (count <= 15) {
        return mp_write_u8(w, MP_FIXMAP | (uint8_t)count);
    } else if (count <= 0xffff) {
        if (mp_write_u8(w, MP_MAP16) < 0) return -1;
        return mp_write_u16be(w, (uint16_t)count);
    } else {
        if (mp_write_u8(w, MP_MAP32) < 0) return -1;
        return mp_write_u32be(w, (uint32_t)count);
    }
}

/* Write array header (caller must write elements after) */
static int mp_write_array(mp_writer_t *w, size_t count)
{
    if (count <= 15) {
        return mp_write_u8(w, MP_FIXARRAY | (uint8_t)count);
    } else if (count <= 0xffff) {
        if (mp_write_u8(w, MP_ARRAY16) < 0) return -1;
        return mp_write_u16be(w, (uint16_t)count);
    } else {
        if (mp_write_u8(w, MP_ARRAY32) < 0) return -1;
        return mp_write_u32be(w, (uint32_t)count);
    }
}

/* =============================================================================
 * MessagePack Reader
 * ============================================================================= */

typedef struct {
    const uint8_t *buf;
    size_t         len;
    size_t         pos;
} mp_reader_t;

static void mp_reader_init(mp_reader_t *r, const uint8_t *buf, size_t len)
{
    r->buf = buf;
    r->len = len;
    r->pos = 0;
}

static int mp_read_u8(mp_reader_t *r, uint8_t *val)
{
    if (r->pos >= r->len) return -1;
    *val = r->buf[r->pos++];
    return 0;
}

static int mp_read_u16be(mp_reader_t *r, uint16_t *val)
{
    if (r->pos + 2 > r->len) return -1;
    *val = (r->buf[r->pos] << 8) | r->buf[r->pos + 1];
    r->pos += 2;
    return 0;
}

static int mp_read_u32be(mp_reader_t *r, uint32_t *val)
{
    if (r->pos + 4 > r->len) return -1;
    *val = ((uint32_t)r->buf[r->pos] << 24) |
           ((uint32_t)r->buf[r->pos + 1] << 16) |
           ((uint32_t)r->buf[r->pos + 2] << 8) |
           (uint32_t)r->buf[r->pos + 3];
    r->pos += 4;
    return 0;
}

/* Read string - returns pointer into buffer, sets len. Does NOT null-terminate. */
static int mp_read_str(mp_reader_t *r, const char **str, size_t *len)
{
    uint8_t marker;
    if (mp_read_u8(r, &marker) < 0) return -1;

    if ((marker & 0xe0) == MP_FIXSTR) {
        *len = marker & 0x1f;
    } else if (marker == MP_STR8) {
        uint8_t l;
        if (mp_read_u8(r, &l) < 0) return -1;
        *len = l;
    } else if (marker == MP_STR16) {
        uint16_t l;
        if (mp_read_u16be(r, &l) < 0) return -1;
        *len = l;
    } else if (marker == MP_STR32) {
        uint32_t l;
        if (mp_read_u32be(r, &l) < 0) return -1;
        *len = l;
    } else {
        return -1;  /* Not a string */
    }

    if (r->pos + *len > r->len) return -1;
    *str = (const char *)&r->buf[r->pos];
    r->pos += *len;
    return 0;
}

/* Read unsigned integer */
static int mp_read_uint(mp_reader_t *r, uint64_t *val)
{
    uint8_t marker;
    if (mp_read_u8(r, &marker) < 0) return -1;

    if (marker <= 0x7f) {
        *val = marker;
        return 0;
    }

    switch (marker) {
        case MP_UINT8: {
            uint8_t v;
            if (mp_read_u8(r, &v) < 0) return -1;
            *val = v;
            return 0;
        }
        case MP_UINT16: {
            uint16_t v;
            if (mp_read_u16be(r, &v) < 0) return -1;
            *val = v;
            return 0;
        }
        case MP_UINT32: {
            uint32_t v;
            if (mp_read_u32be(r, &v) < 0) return -1;
            *val = v;
            return 0;
        }
        case MP_UINT64: {
            uint32_t hi, lo;
            if (mp_read_u32be(r, &hi) < 0) return -1;
            if (mp_read_u32be(r, &lo) < 0) return -1;
            *val = ((uint64_t)hi << 32) | lo;
            return 0;
        }
        default:
            return -1;
    }
}

/* Read map header - returns number of key-value pairs */
static int mp_read_map(mp_reader_t *r, size_t *count)
{
    uint8_t marker;
    if (mp_read_u8(r, &marker) < 0) return -1;

    if ((marker & 0xf0) == MP_FIXMAP) {
        *count = marker & 0x0f;
        return 0;
    }

    switch (marker) {
        case MP_MAP16: {
            uint16_t c;
            if (mp_read_u16be(r, &c) < 0) return -1;
            *count = c;
            return 0;
        }
        case MP_MAP32: {
            uint32_t c;
            if (mp_read_u32be(r, &c) < 0) return -1;
            *count = c;
            return 0;
        }
        default:
            return -1;
    }
}

/* =============================================================================
 * Wire Protocol Functions
 * ============================================================================= */

/*
 * Send a message with length prefix.
 * Format: [4 bytes length BE][payload]
 */
int proto_send(conn_t *conn, const uint8_t *data, size_t len)
{
    if (len > EDB_MAX_MSG_SIZE) {
        LOG("Message too large: %zu", len);
        return -1;
    }

    /* Send length prefix */
    uint32_t len_be = htonl((uint32_t)len);
    if (transport_send(conn->sockfd, (uint8_t *)&len_be, 4) < 0) {
        return -1;
    }

    /* Send payload */
    if (len > 0) {
        if (transport_send(conn->sockfd, data, len) < 0) {
            return -1;
        }
    }

    return 0;
}

/*
 * Receive a message with length prefix.
 * Allocates buffer, caller must free.
 */
int proto_recv(conn_t *conn, uint8_t **data, size_t *len)
{
    uint32_t len_be;

    /* Read length prefix */
    if (transport_recv(conn->sockfd, (uint8_t *)&len_be, 4) < 0) {
        return -1;
    }

    *len = ntohl(len_be);

    if (*len > EDB_MAX_MSG_SIZE) {
        LOG("Message too large: %zu", *len);
        return -1;
    }

    if (*len == 0) {
        *data = NULL;
        return 0;
    }

    /* Allocate and read payload */
    *data = malloc(*len);
    if (!*data) {
        LOG("Out of memory");
        return -1;
    }

    if (transport_recv(conn->sockfd, *data, *len) < 0) {
        free(*data);
        *data = NULL;
        return -1;
    }

    return 0;
}

/*
 * Send hello message.
 * { "type": "hello", "version": 1, "agent": true }
 */
int proto_send_hello(conn_t *conn)
{
    mp_writer_t w;
    if (mp_writer_init(&w, 64) < 0) return -1;

    /* Write map with 3 entries */
    mp_write_map(&w, 3);

    mp_write_str(&w, "type");
    mp_write_str(&w, "hello");

    mp_write_str(&w, "version");
    mp_write_uint(&w, EDB_VERSION);

    mp_write_str(&w, "agent");
    mp_write_bool(&w, true);

    int ret = proto_send(conn, w.buf, w.len);
    mp_writer_free(&w);
    return ret;
}

/*
 * Send hello_ack message.
 * { "type": "hello_ack", "version": 1, "agent": true }
 */
int proto_send_hello_ack(conn_t *conn)
{
    mp_writer_t w;
    if (mp_writer_init(&w, 64) < 0) return -1;

    mp_write_map(&w, 3);

    mp_write_str(&w, "type");
    mp_write_str(&w, "hello_ack");

    mp_write_str(&w, "version");
    mp_write_uint(&w, EDB_VERSION);

    mp_write_str(&w, "agent");
    mp_write_bool(&w, true);

    int ret = proto_send(conn, w.buf, w.len);
    mp_writer_free(&w);
    return ret;
}

/*
 * Send an error response.
 * { "type": "resp", "id": <id>, "ok": false, "error": "<msg>" }
 */
int proto_send_error(conn_t *conn, uint32_t id, const char *error)
{
    mp_writer_t w;
    if (mp_writer_init(&w, 128) < 0) return -1;

    mp_write_map(&w, 4);

    mp_write_str(&w, "type");
    mp_write_str(&w, "resp");

    mp_write_str(&w, "id");
    mp_write_uint(&w, id);

    mp_write_str(&w, "ok");
    mp_write_bool(&w, false);

    mp_write_str(&w, "error");
    mp_write_str(&w, error);

    int ret = proto_send(conn, w.buf, w.len);
    mp_writer_free(&w);
    return ret;
}

/*
 * Send a success response with data.
 * { "type": "resp", "id": <id>, "ok": true, "data": <raw msgpack> }
 *
 * The data parameter should be pre-encoded MessagePack.
 */
int proto_send_response(conn_t *conn, uint32_t id, bool ok,
                        const uint8_t *data, size_t data_len,
                        const char *error)
{
    mp_writer_t w;
    if (mp_writer_init(&w, 128 + data_len) < 0) return -1;

    int num_fields = 3;  /* type, id, ok */
    if (ok && data) num_fields++;
    if (!ok && error) num_fields++;

    mp_write_map(&w, num_fields);

    mp_write_str(&w, "type");
    mp_write_str(&w, "resp");

    mp_write_str(&w, "id");
    mp_write_uint(&w, id);

    mp_write_str(&w, "ok");
    mp_write_bool(&w, ok);

    if (ok && data) {
        mp_write_str(&w, "data");
        /* Write raw msgpack data directly */
        mp_write_raw(&w, data, data_len);
    }

    if (!ok && error) {
        mp_write_str(&w, "error");
        mp_write_str(&w, error);
    }

    int ret = proto_send(conn, w.buf, w.len);
    mp_writer_free(&w);
    return ret;
}

/*
 * Send a data chunk for file transfer.
 * { "type": "data", "id": <id>, "seq": <seq>, "data": <binary>, "done": <bool> }
 */
int proto_send_data(conn_t *conn, uint32_t id, uint32_t seq,
                    const uint8_t *data, size_t len, bool done)
{
    mp_writer_t w;
    if (mp_writer_init(&w, 128 + len) < 0) return -1;

    mp_write_map(&w, 5);

    mp_write_str(&w, "type");
    mp_write_str(&w, "data");

    mp_write_str(&w, "id");
    mp_write_uint(&w, id);

    mp_write_str(&w, "seq");
    mp_write_uint(&w, seq);

    mp_write_str(&w, "data");
    mp_write_bin(&w, data, len);

    mp_write_str(&w, "done");
    mp_write_bool(&w, done);

    int ret = proto_send(conn, w.buf, w.len);
    mp_writer_free(&w);
    return ret;
}

/* =============================================================================
 * Request Parsing and Dispatch
 * ============================================================================= */

/*
 * Parse an incoming request and dispatch to command handlers.
 *
 * Expected format:
 * { "type": "req", "id": <uint>, "cmd": "<string>", "args": { ... } }
 */
int handle_request(conn_t *conn, const uint8_t *msg, size_t msg_len)
{
    mp_reader_t r;
    mp_reader_init(&r, msg, msg_len);

    /* Read map header */
    size_t map_count;
    if (mp_read_map(&r, &map_count) < 0) {
        LOG("Failed to read map header");
        return proto_send_error(conn, 0, "invalid message format");
    }

    /* Parse fields */
    const char *type_str = NULL;
    size_t type_len = 0;
    uint64_t id = 0;
    const char *cmd_str = NULL;
    size_t cmd_len = 0;
    size_t args_start = 0;
    size_t args_len = 0;

    for (size_t i = 0; i < map_count; i++) {
        /* Read key */
        const char *key;
        size_t key_len;
        if (mp_read_str(&r, &key, &key_len) < 0) {
            LOG("Failed to read key");
            return proto_send_error(conn, 0, "invalid message format");
        }

        /* Match key and read value */
        if (key_len == 4 && memcmp(key, "type", 4) == 0) {
            if (mp_read_str(&r, &type_str, &type_len) < 0) {
                return proto_send_error(conn, 0, "invalid type field");
            }
        } else if (key_len == 2 && memcmp(key, "id", 2) == 0) {
            if (mp_read_uint(&r, &id) < 0) {
                return proto_send_error(conn, 0, "invalid id field");
            }
        } else if (key_len == 3 && memcmp(key, "cmd", 3) == 0) {
            if (mp_read_str(&r, &cmd_str, &cmd_len) < 0) {
                return proto_send_error(conn, 0, "invalid cmd field");
            }
        } else if (key_len == 4 && memcmp(key, "args", 4) == 0) {
            /* Save position of args for later parsing by command handler */
            args_start = r.pos;
            /* Skip over the args value - we need to skip whatever type it is */
            /* For simplicity, just pass the rest of the buffer */
            args_len = msg_len - r.pos;
            r.pos = msg_len;  /* Move to end */
        } else {
            /* Skip unknown field - for now just fail */
            LOG("Unknown field in request");
            return proto_send_error(conn, (uint32_t)id, "unknown field");
        }
    }

    /* Validate we got a request */
    if (type_str == NULL || type_len != 3 || memcmp(type_str, "req", 3) != 0) {
        LOG("Not a request message");
        return proto_send_error(conn, (uint32_t)id, "expected request");
    }

    if (cmd_str == NULL) {
        LOG("No command in request");
        return proto_send_error(conn, (uint32_t)id, "missing command");
    }

    /* Null-terminate the command for lookup */
    char cmd_buf[64];
    if (cmd_len >= sizeof(cmd_buf)) {
        return proto_send_error(conn, (uint32_t)id, "command too long");
    }
    memcpy(cmd_buf, cmd_str, cmd_len);
    cmd_buf[cmd_len] = '\0';

    LOG("Request id=%lu cmd=%s", (unsigned long)id, cmd_buf);

    /* Dispatch to command handler */
    cmd_type_t cmd = cmd_parse(cmd_buf);
    return cmd_handle(conn, (uint32_t)id, cmd, msg + args_start, args_len);
}

/* =============================================================================
 * Utility Functions
 * ============================================================================= */

void safe_strcpy(char *dst, const char *src, size_t size)
{
    if (size == 0) return;
    strncpy(dst, src, size - 1);
    dst[size - 1] = '\0';
}
