/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: kill - Send a signal to a process
 *
 * Args:
 *   pid:    int    - Process ID to signal
 *   signal: int    - Signal number (default: 9/SIGKILL)
 */

#define _POSIX_C_SOURCE 200809L

#include <signal.h>
#include <errno.h>
#include <string.h>

#include "edb.h"
#include "commands.h"

int cmd_kill(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    uint64_t pid_val = 0;
    uint64_t sig_val = 9; /* Default: SIGKILL */

    if (parse_uint_arg(args, args_len, "pid", &pid_val) < 0 || pid_val == 0) {
        return proto_send_error(conn, id, "pid is required");
    }

    /* Optional signal argument */
    uint64_t tmp;
    if (parse_uint_arg(args, args_len, "signal", &tmp) == 0) {
        sig_val = tmp;
    }

    LOG("kill: pid=%lu signal=%lu", (unsigned long)pid_val, (unsigned long)sig_val);

    if (kill((pid_t)pid_val, (int)sig_val) < 0) {
        return proto_send_error(conn, id, strerror(errno));
    }

    return proto_send_response(conn, id, true, NULL, 0, NULL);
}
