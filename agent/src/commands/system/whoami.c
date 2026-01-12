/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: whoami - Get current user
 */

#include <stdio.h>
#include <sys/types.h>
#include <unistd.h>
#include <pwd.h>

#include "edb.h"
#include "commands.h"

int cmd_whoami(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    uid_t uid = getuid();
    gid_t gid = getgid();
    struct passwd *pw = getpwuid(uid);

    resp_builder_t rb;
    if (rb_init(&rb, 128) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 3);

    rb_str(&rb, "user");
    rb_str(&rb, pw ? pw->pw_name : "unknown");

    rb_str(&rb, "uid");
    rb_uint(&rb, (uint64_t)uid);

    rb_str(&rb, "gid");
    rb_uint(&rb, (uint64_t)gid);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
