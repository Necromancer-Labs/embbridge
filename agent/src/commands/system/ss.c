/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: ss - Socket statistics (network connections)
 */

#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <dirent.h>
#include <ctype.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* TCP states from include/net/tcp_states.h */
static const char *tcp_state_str(int state)
{
    static const char *states[] = {
        "UNKNOWN",      /* 0 */
        "ESTABLISHED",  /* 1 */
        "SYN_SENT",     /* 2 */
        "SYN_RECV",     /* 3 */
        "FIN_WAIT1",    /* 4 */
        "FIN_WAIT2",    /* 5 */
        "TIME_WAIT",    /* 6 */
        "CLOSE",        /* 7 */
        "CLOSE_WAIT",   /* 8 */
        "LAST_ACK",     /* 9 */
        "LISTEN",       /* 10 */
        "CLOSING",      /* 11 */
    };
    if (state >= 0 && state < (int)(sizeof(states)/sizeof(states[0]))) {
        return states[state];
    }
    return "UNKNOWN";
}

/* Connection info */
typedef struct {
    char     proto[8];
    char     local_addr[64];
    uint16_t local_port;
    char     remote_addr[64];
    uint16_t remote_port;
    char     state[16];
    unsigned long inode;
    int      pid;
    char     process[64];
} conn_info_t;

/* Inode to PID mapping */
typedef struct {
    unsigned long inode;
    int           pid;
    char          name[64];
} inode_map_t;

/* Build inode->pid map by scanning /proc/[pid]/fd/ */
static inode_map_t *build_inode_map(size_t *count_out)
{
    DIR *proc;
    struct dirent *pid_ent;
    inode_map_t *map = NULL;
    size_t count = 0;
    size_t cap = 256;

    map = malloc(cap * sizeof(inode_map_t));
    if (!map) return NULL;

    proc = opendir("/proc");
    if (!proc) {
        free(map);
        return NULL;
    }

    while ((pid_ent = readdir(proc)) != NULL) {
        /* Skip non-numeric entries */
        if (!isdigit((unsigned char)pid_ent->d_name[0])) continue;

        int pid = atoi(pid_ent->d_name);
        if (pid <= 0) continue;

        /* Read process name from /proc/[pid]/comm */
        char comm_path[64];
        char comm[64] = "";
        snprintf(comm_path, sizeof(comm_path), "/proc/%d/comm", pid);
        FILE *cf = fopen(comm_path, "r");
        if (cf) {
            if (fgets(comm, sizeof(comm), cf)) {
                /* Remove trailing newline */
                size_t len = strlen(comm);
                if (len > 0 && comm[len-1] == '\n') comm[len-1] = '\0';
            }
            fclose(cf);
        }

        /* Scan /proc/[pid]/fd/ for socket inodes */
        char fd_path[64];
        snprintf(fd_path, sizeof(fd_path), "/proc/%d/fd", pid);
        DIR *fd_dir = opendir(fd_path);
        if (!fd_dir) continue;

        struct dirent *fd_ent;
        while ((fd_ent = readdir(fd_dir)) != NULL) {
            char link_path[512];
            char link_target[128];
            snprintf(link_path, sizeof(link_path), "%s/%s", fd_path, fd_ent->d_name);

            ssize_t len = readlink(link_path, link_target, sizeof(link_target) - 1);
            if (len <= 0) continue;
            link_target[len] = '\0';

            /* Check if it's a socket: "socket:[12345]" */
            unsigned long inode;
            if (sscanf(link_target, "socket:[%lu]", &inode) == 1) {
                /* Grow map if needed */
                if (count >= cap) {
                    cap *= 2;
                    inode_map_t *newmap = realloc(map, cap * sizeof(inode_map_t));
                    if (!newmap) {
                        closedir(fd_dir);
                        closedir(proc);
                        free(map);
                        return NULL;
                    }
                    map = newmap;
                }
                map[count].inode = inode;
                map[count].pid = pid;
                strncpy(map[count].name, comm, sizeof(map[count].name) - 1);
                map[count].name[sizeof(map[count].name) - 1] = '\0';
                count++;
            }
        }
        closedir(fd_dir);
    }
    closedir(proc);

    *count_out = count;
    return map;
}

/* Find PID for an inode in the map */
static void find_pid_for_inode(const inode_map_t *map, size_t map_count,
                               unsigned long inode, int *pid_out, char *name_out)
{
    *pid_out = 0;
    name_out[0] = '\0';

    for (size_t i = 0; i < map_count; i++) {
        if (map[i].inode == inode) {
            *pid_out = map[i].pid;
            strcpy(name_out, map[i].name);
            return;
        }
    }
}

/* Parse hex IPv4 address to string */
static void parse_ipv4(const char *hex, char *out, size_t out_size)
{
    unsigned int addr;
    if (sscanf(hex, "%X", &addr) != 1) {
        snprintf(out, out_size, "0.0.0.0");
        return;
    }
    /* Linux stores in host byte order (little-endian on most) */
    snprintf(out, out_size, "%u.%u.%u.%u",
             addr & 0xFF,
             (addr >> 8) & 0xFF,
             (addr >> 16) & 0xFF,
             (addr >> 24) & 0xFF);
}

/* Parse hex IPv6 address to string */
static void parse_ipv6(const char *hex, char *out, size_t out_size)
{
    /* IPv6 is stored as 4 32-bit words in hex */
    unsigned int a, b, c, d;
    if (sscanf(hex, "%8X%8X%8X%8X", &a, &b, &c, &d) != 4) {
        snprintf(out, out_size, "::");
        return;
    }

    /* Check for IPv4-mapped address */
    if (a == 0 && b == 0 && c == 0x0000FFFF) {
        snprintf(out, out_size, "::ffff:%u.%u.%u.%u",
                 d & 0xFF, (d >> 8) & 0xFF,
                 (d >> 16) & 0xFF, (d >> 24) & 0xFF);
        return;
    }

    /* Check for all zeros */
    if (a == 0 && b == 0 && c == 0 && d == 0) {
        snprintf(out, out_size, "::");
        return;
    }

    /* Full IPv6 - simplified output */
    snprintf(out, out_size, "%x:%x:%x:%x:%x:%x:%x:%x",
             (a >> 16) & 0xFFFF, a & 0xFFFF,
             (b >> 16) & 0xFFFF, b & 0xFFFF,
             (c >> 16) & 0xFFFF, c & 0xFFFF,
             (d >> 16) & 0xFFFF, d & 0xFFFF);
}

/* Parse /proc/net/{tcp,udp} file */
static int parse_net_file(const char *path, const char *proto, int is_tcp, int is_ipv6,
                          const inode_map_t *inode_map, size_t map_count,
                          conn_info_t **conns, size_t *nconns, size_t *cap)
{
    FILE *f = fopen(path, "r");
    if (!f) return 0;  /* File may not exist, that's OK */

    char line[512];
    int first = 1;

    while (fgets(line, sizeof(line), f)) {
        /* Skip header line */
        if (first) { first = 0; continue; }

        /* Parse line:
         * sl local_address rem_address st tx_queue:rx_queue tr:tm->when retrnsmt uid timeout inode
         */
        unsigned int sl, state;
        char local_addr_hex[65], remote_addr_hex[65];
        unsigned int local_port, remote_port;
        unsigned long inode;

        int matched;
        if (is_ipv6) {
            matched = sscanf(line,
                "%u: %64[0-9A-Fa-f]:%X %64[0-9A-Fa-f]:%X %X %*s %*s %*s %*u %*u %lu",
                &sl, local_addr_hex, &local_port, remote_addr_hex, &remote_port,
                &state, &inode);
        } else {
            matched = sscanf(line,
                "%u: %8[0-9A-Fa-f]:%X %8[0-9A-Fa-f]:%X %X %*s %*s %*s %*u %*u %lu",
                &sl, local_addr_hex, &local_port, remote_addr_hex, &remote_port,
                &state, &inode);
        }

        if (matched < 7) continue;

        /* Grow array if needed */
        if (*nconns >= *cap) {
            *cap *= 2;
            conn_info_t *newconns = realloc(*conns, *cap * sizeof(conn_info_t));
            if (!newconns) {
                fclose(f);
                return -1;
            }
            *conns = newconns;
        }

        conn_info_t *c = &(*conns)[*nconns];
        memset(c, 0, sizeof(*c));

        strncpy(c->proto, proto, sizeof(c->proto) - 1);
        c->local_port = (uint16_t)local_port;
        c->remote_port = (uint16_t)remote_port;
        c->inode = inode;

        if (is_ipv6) {
            parse_ipv6(local_addr_hex, c->local_addr, sizeof(c->local_addr));
            parse_ipv6(remote_addr_hex, c->remote_addr, sizeof(c->remote_addr));
        } else {
            parse_ipv4(local_addr_hex, c->local_addr, sizeof(c->local_addr));
            parse_ipv4(remote_addr_hex, c->remote_addr, sizeof(c->remote_addr));
        }

        if (is_tcp) {
            strncpy(c->state, tcp_state_str((int)state), sizeof(c->state) - 1);
        } else {
            strncpy(c->state, "-", sizeof(c->state) - 1);
        }

        /* Look up PID */
        find_pid_for_inode(inode_map, map_count, inode, &c->pid, c->process);

        (*nconns)++;
    }

    fclose(f);
    return 0;
}

int cmd_netstat(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    /* Build inode->pid map first */
    size_t map_count = 0;
    inode_map_t *inode_map = build_inode_map(&map_count);
    /* inode_map can be NULL if we lack permissions, continue anyway */

    /* Collect connections */
    conn_info_t *conns = NULL;
    size_t nconns = 0;
    size_t cap = 128;

    conns = malloc(cap * sizeof(conn_info_t));
    if (!conns) {
        free(inode_map);
        return proto_send_error(conn, id, "out of memory");
    }

    /* Parse all network files */
    parse_net_file("/proc/net/tcp",  "tcp",  1, 0, inode_map, map_count, &conns, &nconns, &cap);
    parse_net_file("/proc/net/tcp6", "tcp6", 1, 1, inode_map, map_count, &conns, &nconns, &cap);
    parse_net_file("/proc/net/udp",  "udp",  0, 0, inode_map, map_count, &conns, &nconns, &cap);
    parse_net_file("/proc/net/udp6", "udp6", 0, 1, inode_map, map_count, &conns, &nconns, &cap);

    free(inode_map);

    LOG("ss: found %zu connections", nconns);

    /* Build response */
    resp_builder_t rb;
    if (rb_init(&rb, 4096) < 0) {
        free(conns);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "connections");
    rb_array(&rb, nconns);

    for (size_t i = 0; i < nconns; i++) {
        conn_info_t *c = &conns[i];
        rb_map(&rb, 8);

        rb_str(&rb, "proto");
        rb_str(&rb, c->proto);

        rb_str(&rb, "local_addr");
        rb_str(&rb, c->local_addr);

        rb_str(&rb, "local_port");
        rb_uint(&rb, c->local_port);

        rb_str(&rb, "remote_addr");
        rb_str(&rb, c->remote_addr);

        rb_str(&rb, "remote_port");
        rb_uint(&rb, c->remote_port);

        rb_str(&rb, "state");
        rb_str(&rb, c->state);

        rb_str(&rb, "pid");
        rb_uint(&rb, (uint64_t)c->pid);

        rb_str(&rb, "process");
        rb_str(&rb, c->process);
    }

    free(conns);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
