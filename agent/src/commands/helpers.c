/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command helpers: argument parsing, response building, path utilities.
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#include "commands.h"

/* =============================================================================
 * Response Builder Implementation
 * ============================================================================= */

static int rb_ensure(resp_builder_t *rb, size_t need)
{
    if (rb->len + need <= rb->cap) return 0;
    size_t new_cap = rb->cap * 2;
    while (new_cap < rb->len + need) new_cap *= 2;
    uint8_t *new_buf = realloc(rb->buf, new_cap);
    if (!new_buf) return -1;
    rb->buf = new_buf;
    rb->cap = new_cap;
    return 0;
}

int rb_init(resp_builder_t *rb, size_t cap)
{
    rb->buf = malloc(cap);
    if (!rb->buf) return -1;
    rb->cap = cap;
    rb->len = 0;
    return 0;
}

void rb_free(resp_builder_t *rb)
{
    free(rb->buf);
    rb->buf = NULL;
}

int rb_u8(resp_builder_t *rb, uint8_t v)
{
    if (rb_ensure(rb, 1) < 0) return -1;
    rb->buf[rb->len++] = v;
    return 0;
}

int rb_u16be(resp_builder_t *rb, uint16_t v)
{
    if (rb_ensure(rb, 2) < 0) return -1;
    rb->buf[rb->len++] = (v >> 8) & 0xff;
    rb->buf[rb->len++] = v & 0xff;
    return 0;
}

int rb_u32be(resp_builder_t *rb, uint32_t v)
{
    if (rb_ensure(rb, 4) < 0) return -1;
    rb->buf[rb->len++] = (v >> 24) & 0xff;
    rb->buf[rb->len++] = (v >> 16) & 0xff;
    rb->buf[rb->len++] = (v >> 8) & 0xff;
    rb->buf[rb->len++] = v & 0xff;
    return 0;
}

int rb_raw(resp_builder_t *rb, const void *data, size_t len)
{
    if (rb_ensure(rb, len) < 0) return -1;
    memcpy(rb->buf + rb->len, data, len);
    rb->len += len;
    return 0;
}

int rb_str(resp_builder_t *rb, const char *s)
{
    size_t len = strlen(s);
    if (len <= 31) {
        if (rb_u8(rb, 0xa0 | (uint8_t)len) < 0) return -1;
    } else if (len <= 0xff) {
        if (rb_u8(rb, 0xd9) < 0) return -1;
        if (rb_u8(rb, (uint8_t)len) < 0) return -1;
    } else {
        if (rb_u8(rb, 0xda) < 0) return -1;
        if (rb_u16be(rb, (uint16_t)len) < 0) return -1;
    }
    return rb_raw(rb, s, len);
}

int rb_bin(resp_builder_t *rb, const uint8_t *data, size_t len)
{
    /* MessagePack binary format:
     * bin8:  0xc4 + 1 byte length + data
     * bin16: 0xc5 + 2 bytes length + data
     * bin32: 0xc6 + 4 bytes length + data
     */
    if (len <= 0xff) {
        if (rb_u8(rb, 0xc4) < 0) return -1;
        if (rb_u8(rb, (uint8_t)len) < 0) return -1;
    } else if (len <= 0xffff) {
        if (rb_u8(rb, 0xc5) < 0) return -1;
        if (rb_u16be(rb, (uint16_t)len) < 0) return -1;
    } else {
        if (rb_u8(rb, 0xc6) < 0) return -1;
        if (rb_u32be(rb, (uint32_t)len) < 0) return -1;
    }
    return rb_raw(rb, data, len);
}

int rb_uint(resp_builder_t *rb, uint64_t v)
{
    if (v <= 0x7f) {
        return rb_u8(rb, (uint8_t)v);
    } else if (v <= 0xff) {
        if (rb_u8(rb, 0xcc) < 0) return -1;
        return rb_u8(rb, (uint8_t)v);
    } else if (v <= 0xffff) {
        if (rb_u8(rb, 0xcd) < 0) return -1;
        return rb_u16be(rb, (uint16_t)v);
    } else {
        if (rb_u8(rb, 0xce) < 0) return -1;
        return rb_u32be(rb, (uint32_t)v);
    }
}

int rb_map(resp_builder_t *rb, size_t count)
{
    if (count <= 15) {
        return rb_u8(rb, 0x80 | (uint8_t)count);
    } else {
        if (rb_u8(rb, 0xde) < 0) return -1;
        return rb_u16be(rb, (uint16_t)count);
    }
}

int rb_array(resp_builder_t *rb, size_t count)
{
    if (count <= 15) {
        return rb_u8(rb, 0x90 | (uint8_t)count);
    } else {
        if (rb_u8(rb, 0xdc) < 0) return -1;
        return rb_u16be(rb, (uint16_t)count);
    }
}

/* =============================================================================
 * Argument Parsing
 *
 * MessagePack format reference (subset we use):
 *   - fixmap:  0x80-0x8f (up to 15 key-value pairs)
 *   - map16:   0xde + 2 bytes length
 *   - fixstr:  0xa0-0xbf (up to 31 bytes)
 *   - str8:    0xd9 + 1 byte length
 *   - str16:   0xda + 2 bytes length
 *   - fixint:  0x00-0x7f (0 to 127)
 *   - uint8:   0xcc + 1 byte
 *   - uint16:  0xcd + 2 bytes
 *   - uint32:  0xce + 4 bytes
 *   - uint64:  0xcf + 8 bytes
 *   - true:    0xc3
 *   - false:   0xc2
 *   - nil:     0xc0
 * ============================================================================= */

char *parse_string_arg(const uint8_t *args, size_t args_len, const char *key)
{
    if (!args || args_len == 0) return NULL;

    size_t pos = 0;
    size_t key_len = strlen(key);

    /* Read map header */
    if (pos >= args_len) return NULL;
    uint8_t marker = args[pos++];

    size_t map_count;
    if ((marker & 0xf0) == 0x80) {
        map_count = marker & 0x0f;
    } else if (marker == 0xde) {
        if (pos + 2 > args_len) return NULL;
        map_count = (args[pos] << 8) | args[pos + 1];
        pos += 2;
    } else {
        return NULL;
    }

    /* Iterate through key-value pairs */
    for (size_t i = 0; i < map_count; i++) {
        /* Read key string */
        if (pos >= args_len) return NULL;
        uint8_t km = args[pos++];

        size_t klen;
        if ((km & 0xe0) == 0xa0) {
            klen = km & 0x1f;
        } else if (km == 0xd9) {
            if (pos >= args_len) return NULL;
            klen = args[pos++];
        } else if (km == 0xda) {
            if (pos + 2 > args_len) return NULL;
            klen = (args[pos] << 8) | args[pos + 1];
            pos += 2;
        } else {
            return NULL;
        }

        if (pos + klen > args_len) return NULL;
        const char *kstr = (const char *)&args[pos];
        pos += klen;

        /* Check if this is our key */
        bool is_target = (klen == key_len && memcmp(kstr, key, klen) == 0);

        /* Read value */
        if (pos >= args_len) return NULL;
        uint8_t vm = args[pos++];

        size_t vlen;
        if ((vm & 0xe0) == 0xa0) {
            /* fixstr */
            vlen = vm & 0x1f;
            if (pos + vlen > args_len) return NULL;
            if (is_target) {
                char *result = malloc(vlen + 1);
                if (!result) return NULL;
                memcpy(result, &args[pos], vlen);
                result[vlen] = '\0';
                return result;
            }
            pos += vlen;
        } else if (vm == 0xd9) {
            /* str8 */
            if (pos >= args_len) return NULL;
            vlen = args[pos++];
            if (pos + vlen > args_len) return NULL;
            if (is_target) {
                char *result = malloc(vlen + 1);
                if (!result) return NULL;
                memcpy(result, &args[pos], vlen);
                result[vlen] = '\0';
                return result;
            }
            pos += vlen;
        } else if (vm == 0xda) {
            /* str16 */
            if (pos + 2 > args_len) return NULL;
            vlen = (args[pos] << 8) | args[pos + 1];
            pos += 2;
            if (pos + vlen > args_len) return NULL;
            if (is_target) {
                char *result = malloc(vlen + 1);
                if (!result) return NULL;
                memcpy(result, &args[pos], vlen);
                result[vlen] = '\0';
                return result;
            }
            pos += vlen;
        } else if (vm <= 0x7f) {
            /* positive fixint - skip */
        } else if (vm == 0xcc) {
            pos++;
        } else if (vm == 0xcd) {
            pos += 2;
        } else if (vm == 0xce) {
            pos += 4;
        } else if (vm == 0xcf) {
            pos += 8;
        } else if (vm == 0xc2 || vm == 0xc3) {
            /* false/true */
        } else if (vm == 0xc0) {
            /* nil */
        } else {
            return NULL;
        }
    }

    return NULL;
}

int parse_uint_arg(const uint8_t *args, size_t args_len, const char *key, uint64_t *out)
{
    if (!args || args_len == 0) return -1;

    size_t pos = 0;
    size_t key_len = strlen(key);

    /* Read map header */
    if (pos >= args_len) return -1;
    uint8_t marker = args[pos++];

    size_t map_count;
    if ((marker & 0xf0) == 0x80) {
        map_count = marker & 0x0f;
    } else if (marker == 0xde) {
        if (pos + 2 > args_len) return -1;
        map_count = (args[pos] << 8) | args[pos + 1];
        pos += 2;
    } else {
        return -1;
    }

    /* Search for key */
    for (size_t i = 0; i < map_count; i++) {
        /* Read key string */
        if (pos >= args_len) return -1;
        uint8_t km = args[pos++];

        size_t klen;
        if ((km & 0xe0) == 0xa0) {
            klen = km & 0x1f;
        } else if (km == 0xd9) {
            if (pos >= args_len) return -1;
            klen = args[pos++];
        } else if (km == 0xda) {
            if (pos + 2 > args_len) return -1;
            klen = (args[pos] << 8) | args[pos + 1];
            pos += 2;
        } else {
            return -1;
        }

        if (pos + klen > args_len) return -1;
        const char *kstr = (const char *)&args[pos];
        pos += klen;

        bool is_target = (klen == key_len && memcmp(kstr, key, klen) == 0);

        /* Read value */
        if (pos >= args_len) return -1;
        uint8_t vm = args[pos++];

        if (vm <= 0x7f) {
            if (is_target) {
                *out = vm;
                return 0;
            }
        } else if (vm == 0xcc) {
            if (pos >= args_len) return -1;
            if (is_target) {
                *out = args[pos];
                return 0;
            }
            pos++;
        } else if (vm == 0xcd) {
            if (pos + 2 > args_len) return -1;
            if (is_target) {
                *out = (args[pos] << 8) | args[pos + 1];
                return 0;
            }
            pos += 2;
        } else if (vm == 0xce) {
            if (pos + 4 > args_len) return -1;
            if (is_target) {
                *out = ((uint64_t)args[pos] << 24) |
                       ((uint64_t)args[pos + 1] << 16) |
                       ((uint64_t)args[pos + 2] << 8) |
                       (uint64_t)args[pos + 3];
                return 0;
            }
            pos += 4;
        } else if (vm == 0xcf) {
            if (pos + 8 > args_len) return -1;
            if (is_target) {
                *out = ((uint64_t)args[pos] << 56) |
                       ((uint64_t)args[pos + 1] << 48) |
                       ((uint64_t)args[pos + 2] << 40) |
                       ((uint64_t)args[pos + 3] << 32) |
                       ((uint64_t)args[pos + 4] << 24) |
                       ((uint64_t)args[pos + 5] << 16) |
                       ((uint64_t)args[pos + 6] << 8) |
                       (uint64_t)args[pos + 7];
                return 0;
            }
            pos += 8;
        } else if ((vm & 0xe0) == 0xa0) {
            pos += (vm & 0x1f);
        } else if (vm == 0xd9) {
            if (pos >= args_len) return -1;
            size_t slen = args[pos++];
            pos += slen;
        } else if (vm == 0xda) {
            if (pos + 2 > args_len) return -1;
            size_t slen = (args[pos] << 8) | args[pos + 1];
            pos += 2 + slen;
        } else {
            return -1;
        }
    }

    return -1;
}

/* =============================================================================
 * Path Utilities
 * ============================================================================= */

char *path_resolve(const char *cwd, const char *path)
{
    char *result;

    if (path[0] == '/') {
        result = strdup(path);
    } else {
        size_t cwd_len = strlen(cwd);
        size_t path_len = strlen(path);
        size_t total = cwd_len + 1 + path_len + 1;

        result = malloc(total);
        if (!result) return NULL;

        if (cwd_len > 0 && cwd[cwd_len - 1] == '/') {
            snprintf(result, total, "%s%s", cwd, path);
        } else {
            snprintf(result, total, "%s/%s", cwd, path);
        }
    }

    return result;
}

bool path_is_dir(const char *path)
{
    struct stat st;
    if (stat(path, &st) < 0) return false;
    return S_ISDIR(st.st_mode);
}

bool path_exists(const char *path)
{
    struct stat st;
    return stat(path, &st) == 0;
}
