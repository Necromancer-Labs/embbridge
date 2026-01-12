/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: reboot - Reboot the system
 */

#define _GNU_SOURCE

#include <stdio.h>
#include <unistd.h>
#include <sys/reboot.h>
#include <errno.h>
#include <string.h>

#include "edb.h"
#include "commands.h"

int cmd_reboot(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    LOG("Rebooting system...");

    /* Send response before reboot */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "status");
    rb_str(&rb, "rebooting");

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);

    if (ret < 0) {
        return ret;
    }

    /* Sync filesystems */
    sync();

    /* Reboot the system */
    reboot(RB_AUTOBOOT);

    /* If we get here, reboot failed */
    return proto_send_error(conn, id, strerror(errno));
}
