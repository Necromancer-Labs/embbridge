/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File transfer commands: get (download), put (upload)
 * These handle chunked file transfers for large files.
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* =============================================================================
 * Command: get (download file from device)
 *
 * Protocol:
 *   1. Client sends: { cmd: "get", args: { path: "/path/to/file" } }
 *   2. Agent sends:  { ok: true, data: { size: N, mode: M } }
 *   3. Agent sends:  { type: "data", seq: 0, data: <chunk>, done: false }
 *   4. Agent sends:  { type: "data", seq: 1, data: <chunk>, done: false }
 *   5. ...
 *   6. Agent sends:  { type: "data", seq: N, data: <chunk>, done: true }
 *
 * Files are sent in 64KB chunks to avoid memory issues on constrained devices.
 * ============================================================================= */

int cmd_get(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Open the file */
    FILE *f = fopen(resolved, "rb");
    if (!f) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    /* Get file info */
    struct stat st;
    if (fstat(fileno(f), &st) < 0) {
        int err = errno;
        fclose(f);
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }
    free(resolved);

    if (S_ISDIR(st.st_mode)) {
        fclose(f);
        return proto_send_error(conn, id, "is a directory");
    }

    uint64_t file_size = (uint64_t)st.st_size;
    uint32_t file_mode = (uint32_t)(st.st_mode & 0777);

    LOG("get: sending file, size=%lu, mode=%o", (unsigned long)file_size, file_mode);

    /* Send initial response with file info */
    resp_builder_t rb;
    if (rb_init(&rb, 64) < 0) {
        fclose(f);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 2);
    rb_str(&rb, "size");
    rb_uint(&rb, file_size);
    rb_str(&rb, "mode");
    rb_uint(&rb, file_mode);

    if (proto_send_response(conn, id, true, rb.buf, rb.len, NULL) < 0) {
        rb_free(&rb);
        fclose(f);
        return -1;
    }
    rb_free(&rb);

    /* Send file data in chunks */
    uint8_t chunk[EDB_CHUNK_SIZE];
    uint32_t seq = 0;
    size_t total_sent = 0;

    while (total_sent < file_size) {
        size_t to_read = file_size - total_sent;
        if (to_read > EDB_CHUNK_SIZE) to_read = EDB_CHUNK_SIZE;

        size_t n = fread(chunk, 1, to_read, f);
        if (n == 0) {
            if (ferror(f)) {
                fclose(f);
                return proto_send_error(conn, id, "read error");
            }
            break;  /* EOF */
        }

        total_sent += n;
        bool done = (total_sent >= file_size);

        LOG("get: sending chunk seq=%u, len=%zu, done=%d", seq, n, done);

        if (proto_send_data(conn, id, seq, chunk, n, done) < 0) {
            fclose(f);
            return -1;
        }

        seq++;
    }

    fclose(f);
    LOG("get: transfer complete, sent %zu bytes in %u chunks", total_sent, seq);
    return 0;
}

/* =============================================================================
 * Command: put (upload file to device)
 *
 * Protocol:
 *   1. Client sends: { cmd: "put", args: { path: "/path", size: N, mode: M } }
 *   2. Agent sends:  { ok: true, data: {} }
 *   3. Client sends: { type: "data", seq: 0, data: <chunk>, done: false }
 *   4. Client sends: { type: "data", seq: 1, data: <chunk>, done: false }
 *   5. ...
 *   6. Client sends: { type: "data", seq: N, data: <chunk>, done: true }
 *
 * The agent creates/overwrites the file and writes each chunk as it arrives.
 * ============================================================================= */

int cmd_put(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    uint64_t file_size = 0;
    uint64_t file_mode = 0644;  /* Default mode */

    parse_uint_arg(args, args_len, "size", &file_size);
    parse_uint_arg(args, args_len, "mode", &file_mode);

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    LOG("put: receiving file %s, size=%lu, mode=%o",
        resolved, (unsigned long)file_size, (unsigned int)file_mode);

    /* Open file for writing */
    FILE *f = fopen(resolved, "wb");
    if (!f) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    /* Set file mode */
    fchmod(fileno(f), (mode_t)file_mode);

    /* Send OK response - client will start sending data chunks */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        fclose(f);
        free(resolved);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 0);  /* Empty data map */

    if (proto_send_response(conn, id, true, rb.buf, rb.len, NULL) < 0) {
        rb_free(&rb);
        fclose(f);
        free(resolved);
        return -1;
    }
    rb_free(&rb);

    /* Receive data chunks */
    size_t total_received = 0;
    uint32_t expected_seq = 0;

    while (1) {
        uint8_t *msg = NULL;
        size_t msg_len;

        if (proto_recv(conn, &msg, &msg_len) < 0) {
            LOG("put: failed to receive data chunk");
            fclose(f);
            free(resolved);
            return -1;
        }

        /* Parse the data message */
        /* Expected: { "type": "data", "id": N, "seq": N, "data": <bin>, "done": bool } */
        size_t pos = 0;
        bool done = false;
        const uint8_t *chunk_data = NULL;
        size_t chunk_len = 0;

        /* Skip map header */
        if (pos >= msg_len) goto parse_error;
        uint8_t marker = msg[pos++];
        size_t map_count;
        if ((marker & 0xf0) == 0x80) {
            map_count = marker & 0x0f;
        } else {
            goto parse_error;
        }

        for (size_t i = 0; i < map_count; i++) {
            /* Read key */
            if (pos >= msg_len) goto parse_error;
            uint8_t km = msg[pos++];
            size_t klen = 0;
            if ((km & 0xe0) == 0xa0) {
                klen = km & 0x1f;
            } else {
                goto parse_error;
            }

            if (pos + klen > msg_len) goto parse_error;
            const char *key = (const char *)&msg[pos];
            pos += klen;

            /* Read value based on key */
            if (klen == 4 && memcmp(key, "data", 4) == 0) {
                /* Binary data */
                if (pos >= msg_len) goto parse_error;
                uint8_t vm = msg[pos++];
                if (vm == 0xc4) {
                    /* bin8 */
                    if (pos >= msg_len) goto parse_error;
                    chunk_len = msg[pos++];
                } else if (vm == 0xc5) {
                    /* bin16 */
                    if (pos + 2 > msg_len) goto parse_error;
                    chunk_len = (msg[pos] << 8) | msg[pos + 1];
                    pos += 2;
                } else if (vm == 0xc6) {
                    /* bin32 */
                    if (pos + 4 > msg_len) goto parse_error;
                    chunk_len = ((size_t)msg[pos] << 24) |
                                ((size_t)msg[pos + 1] << 16) |
                                ((size_t)msg[pos + 2] << 8) |
                                (size_t)msg[pos + 3];
                    pos += 4;
                } else {
                    goto parse_error;
                }
                if (pos + chunk_len > msg_len) goto parse_error;
                chunk_data = &msg[pos];
                pos += chunk_len;
            } else if (klen == 4 && memcmp(key, "done", 4) == 0) {
                /* Boolean */
                if (pos >= msg_len) goto parse_error;
                uint8_t vm = msg[pos++];
                done = (vm == 0xc3);  /* true */
            } else {
                /* Skip other values */
                if (pos >= msg_len) goto parse_error;
                uint8_t vm = msg[pos++];
                if (vm <= 0x7f || vm == 0xc2 || vm == 0xc3) {
                    /* fixint, false, true - no extra bytes */
                } else if (vm == 0xcc) {
                    pos++;
                } else if (vm == 0xcd) {
                    pos += 2;
                } else if (vm == 0xce) {
                    pos += 4;
                } else if ((vm & 0xe0) == 0xa0) {
                    pos += (vm & 0x1f);
                } else if (vm == 0xd9) {
                    if (pos >= msg_len) goto parse_error;
                    pos += 1 + msg[pos];
                }
            }
        }

        if (chunk_data && chunk_len > 0) {
            if (fwrite(chunk_data, 1, chunk_len, f) != chunk_len) {
                LOG("put: write error");
                free(msg);
                fclose(f);
                free(resolved);
                return proto_send_error(conn, id, "write error");
            }
            total_received += chunk_len;
            LOG("put: received chunk seq=%u, len=%zu, done=%d",
                expected_seq, chunk_len, done);
        }

        free(msg);
        expected_seq++;

        if (done) break;
        continue;

    parse_error:
        LOG("put: parse error in data chunk");
        free(msg);
        fclose(f);
        free(resolved);
        return proto_send_error(conn, id, "invalid data chunk");
    }

    fclose(f);
    free(resolved);

    LOG("put: transfer complete, received %zu bytes", total_received);
    return 0;
}
