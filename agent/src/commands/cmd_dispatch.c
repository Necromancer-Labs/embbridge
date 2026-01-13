/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command dispatch: parsing command names and routing to handlers.
 */

#include <string.h>
#include "edb.h"

/* =============================================================================
 * Command Table
 * ============================================================================= */

typedef struct {
    const char  *name;
    cmd_type_t   type;
} cmd_entry_t;

static const cmd_entry_t cmd_table[] = {
    { "ls",       CMD_LS },
    { "cat",      CMD_CAT },
    { "pwd",      CMD_PWD },
    { "cd",       CMD_CD },
    { "realpath", CMD_REALPATH },
    { "pull",     CMD_PULL },
    { "push",     CMD_PUSH },
    { "exec",     CMD_EXEC },
    { "mkdir",    CMD_MKDIR },
    { "rm",       CMD_RM },
    { "mv",       CMD_MV },
    { "cp",       CMD_CP },
    { "chmod",    CMD_CHMOD },
    { "touch",    CMD_TOUCH },
    { "uname",    CMD_UNAME },
    { "ps",       CMD_PS },
    { "ss",       CMD_NETSTAT },
    { "env",      CMD_ENV },
    { "mtd",      CMD_MTD },
    { "firmware", CMD_FIRMWARE },
    { "hexdump",    CMD_HEXDUMP },
    { "kill-agent", CMD_KILL_AGENT },
    { "reboot",     CMD_REBOOT },
    { "whoami",     CMD_WHOAMI },
    { "dmesg",      CMD_DMESG },
    { "strings",    CMD_STRINGS },
    { "cpuinfo",    CMD_CPUINFO },
    { "mtd",        CMD_MTD },
    { "ip_addr",    CMD_IP_ADDR },
    { "ip_route",   CMD_IP_ROUTE },
    { NULL,         CMD_UNKNOWN },
};

/* =============================================================================
 * Command Parsing
 * ============================================================================= */

cmd_type_t cmd_parse(const char *name)
{
    for (const cmd_entry_t *e = cmd_table; e->name; e++) {
        if (strcmp(name, e->name) == 0) {
            return e->type;
        }
    }
    return CMD_UNKNOWN;
}

/* =============================================================================
 * Command Dispatch
 * ============================================================================= */

int cmd_handle(conn_t *conn, uint32_t id, cmd_type_t cmd,
               const uint8_t *args, size_t args_len)
{
    switch (cmd) {
        /* Basic commands (basic_commands.c) */
        case CMD_LS:       return cmd_ls(conn, id, args, args_len);
        case CMD_CAT:      return cmd_cat(conn, id, args, args_len);
        case CMD_PWD:      return cmd_pwd(conn, id, args, args_len);
        case CMD_CD:       return cmd_cd(conn, id, args, args_len);
        case CMD_REALPATH: return cmd_realpath(conn, id, args, args_len);

        /* File transfer (file_transfer.c) */
        case CMD_PULL:    return cmd_pull(conn, id, args, args_len);
        case CMD_PUSH:    return cmd_push(conn, id, args, args_len);

        /* File operations (file_operations.c) */
        case CMD_RM:      return cmd_rm(conn, id, args, args_len);
        case CMD_MV:      return cmd_mv(conn, id, args, args_len);
        case CMD_CP:      return cmd_cp(conn, id, args, args_len);
        case CMD_MKDIR:   return cmd_mkdir(conn, id, args, args_len);
        case CMD_CHMOD:   return cmd_chmod(conn, id, args, args_len);
        case CMD_TOUCH:   return cmd_touch(conn, id, args, args_len);

        /* System commands (system_commands.c) */
        case CMD_UNAME:      return cmd_uname(conn, id, args, args_len);
        case CMD_PS:         return cmd_ps(conn, id, args, args_len);
        case CMD_EXEC:       return cmd_exec(conn, id, args, args_len);
        case CMD_NETSTAT:    return cmd_netstat(conn, id, args, args_len);
        case CMD_KILL_AGENT: return cmd_kill_agent(conn, id, args, args_len);
        case CMD_REBOOT:     return cmd_reboot(conn, id, args, args_len);
        case CMD_WHOAMI:     return cmd_whoami(conn, id, args, args_len);
        case CMD_DMESG:      return cmd_dmesg(conn, id, args, args_len);
        case CMD_STRINGS:    return cmd_strings(conn, id, args, args_len);
        case CMD_CPUINFO:    return cmd_cpuinfo(conn, id, args, args_len);
        case CMD_MTD:        return cmd_mtd(conn, id, args, args_len);
        case CMD_IP_ADDR:    return cmd_ip_addr(conn, id, args, args_len);
        case CMD_IP_ROUTE:   return cmd_ip_route(conn, id, args, args_len);

        /* Unimplemented commands */
        case CMD_ENV:
        case CMD_FIRMWARE:
        case CMD_HEXDUMP:
        case CMD_UNKNOWN:
        default:
            return proto_send_error(conn, id, "unknown command");
    }
}
