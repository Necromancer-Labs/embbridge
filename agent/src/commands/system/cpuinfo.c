/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: cpuinfo - Show CPU information
 */

#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>

#include "edb.h"
#include "commands.h"

int cmd_cpuinfo(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    FILE *f = fopen("/proc/cpuinfo", "r");
    if (!f) {
        return proto_send_error(conn, id, strerror(errno));
    }

    /* Read entire file */
    size_t capacity = 4096;
    size_t len = 0;
    char *buf = malloc(capacity);
    if (!buf) {
        fclose(f);
        return proto_send_error(conn, id, "out of memory");
    }

    size_t n;
    while ((n = fread(buf + len, 1, capacity - len, f)) > 0) {
        len += n;
        if (len >= capacity) {
            capacity *= 2;
            char *newbuf = realloc(buf, capacity);
            if (!newbuf) {
                free(buf);
                fclose(f);
                return proto_send_error(conn, id, "out of memory");
            }
            buf = newbuf;
        }
    }

    fclose(f);

    LOG("cpuinfo: read %zu bytes", len);

    resp_builder_t rb;
    if (rb_init(&rb, len + 64) < 0) {
        free(buf);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "content");
    rb_bin(&rb, (uint8_t *)buf, len);

    free(buf);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
