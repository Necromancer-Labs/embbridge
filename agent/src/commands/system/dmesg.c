/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: dmesg - Read kernel log messages
 */

#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <sys/klog.h>
#include <errno.h>
#include <string.h>

#include "edb.h"
#include "commands.h"

int cmd_dmesg(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    /* Get the size of the kernel log buffer */
    int bufsize = klogctl(10, NULL, 0);  /* SYSLOG_ACTION_SIZE_BUFFER */
    if (bufsize < 0) {
        return proto_send_error(conn, id, strerror(errno));
    }

    if (bufsize == 0) {
        bufsize = 16384;  /* Default fallback */
    }

    char *buf = malloc((size_t)bufsize);
    if (!buf) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Read the kernel log */
    int len = klogctl(3, buf, bufsize);  /* SYSLOG_ACTION_READ_ALL */
    if (len < 0) {
        int err = errno;
        free(buf);
        return proto_send_error(conn, id, strerror(err));
    }

    LOG("dmesg: read %d bytes from kernel log", len);

    resp_builder_t rb;
    if (rb_init(&rb, (size_t)len + 64) < 0) {
        free(buf);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "log");
    rb_bin(&rb, (uint8_t *)buf, (size_t)len);

    free(buf);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
