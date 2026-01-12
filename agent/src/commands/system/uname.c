/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: uname - Get system information
 */

#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <sys/utsname.h>

#include "edb.h"
#include "commands.h"

int cmd_uname(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    struct utsname uts;
    if (uname(&uts) < 0) {
        return proto_send_error(conn, id, strerror(errno));
    }

    resp_builder_t rb;
    if (rb_init(&rb, 512) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 5);

    rb_str(&rb, "sysname");
    rb_str(&rb, uts.sysname);

    rb_str(&rb, "nodename");
    rb_str(&rb, uts.nodename);

    rb_str(&rb, "release");
    rb_str(&rb, uts.release);

    rb_str(&rb, "version");
    rb_str(&rb, uts.version);

    rb_str(&rb, "machine");
    rb_str(&rb, uts.machine);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
