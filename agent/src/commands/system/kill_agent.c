/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: kill-agent - Kill the parent agent process because we fork the client connection as a child.
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <sys/types.h>
#include <unistd.h>
#include <signal.h>
#include <errno.h>
#include <string.h>

#include "edb.h"
#include "commands.h"

int cmd_kill_agent(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    pid_t ppid = getppid();

    /* Check if we're a forked child (ppid != 1 means we have a parent) */
    if (ppid <= 1) {
        return proto_send_error(conn, id, "not running in fork mode (no parent to kill)");
    }

    LOG("Killing parent agent (pid %d)", ppid);

    /* Send SIGTERM to parent */
    if (kill(ppid, SIGTERM) < 0) {
        return proto_send_error(conn, id, strerror(errno));
    }

    /* Send success response before we potentially get killed too */
    resp_builder_t rb;
    if (rb_init(&rb, 64) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "killed_pid");
    rb_uint(&rb, (uint64_t)ppid);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
