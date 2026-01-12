/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: ps - List processes
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <ctype.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* Process info structure */
typedef struct {
    int      pid;
    int      ppid;
    char     name[256];
    char     state;
    char     cmdline[1024];
} proc_info_t;

/* Read a single process's info from /proc/[pid] */
static int read_proc_info(int pid, proc_info_t *info)
{
    char path[64];
    FILE *f;
    char buf[2048];

    info->pid = pid;
    info->ppid = 0;
    info->name[0] = '\0';
    info->state = '?';
    info->cmdline[0] = '\0';

    /* Read /proc/[pid]/stat for basic info */
    snprintf(path, sizeof(path), "/proc/%d/stat", pid);
    f = fopen(path, "r");
    if (!f) {
        return -1;
    }

    if (fgets(buf, sizeof(buf), f)) {
        /* Format: pid (comm) state ppid ...
         * comm can contain spaces and parentheses, so find last ')' */
        char *start = strchr(buf, '(');
        char *end = strrchr(buf, ')');

        if (start && end && end > start) {
            /* Extract name (between parentheses) */
            size_t namelen = (size_t)(end - start - 1);
            if (namelen >= sizeof(info->name)) {
                namelen = sizeof(info->name) - 1;
            }
            memcpy(info->name, start + 1, namelen);
            info->name[namelen] = '\0';

            /* Parse state and ppid after the ')' */
            char state;
            int ppid;
            if (sscanf(end + 2, "%c %d", &state, &ppid) == 2) {
                info->state = state;
                info->ppid = ppid;
            }
        }
    }
    fclose(f);

    /* Read /proc/[pid]/cmdline for full command */
    snprintf(path, sizeof(path), "/proc/%d/cmdline", pid);
    f = fopen(path, "r");
    if (f) {
        size_t n = fread(buf, 1, sizeof(buf) - 1, f);
        fclose(f);

        if (n > 0) {
            /* cmdline has NUL separators between args, replace with spaces */
            for (size_t i = 0; i < n; i++) {
                if (buf[i] == '\0') {
                    buf[i] = ' ';
                }
            }
            /* Trim trailing space */
            while (n > 0 && buf[n - 1] == ' ') {
                n--;
            }
            buf[n] = '\0';

            /* Copy to info, truncate if needed */
            size_t len = n;
            if (len >= sizeof(info->cmdline)) {
                len = sizeof(info->cmdline) - 1;
            }
            memcpy(info->cmdline, buf, len);
            info->cmdline[len] = '\0';
        }
    }

    /* If no cmdline (kernel thread), use [name] */
    if (info->cmdline[0] == '\0' && info->name[0] != '\0') {
        snprintf(info->cmdline, sizeof(info->cmdline), "[%s]", info->name);
    }

    return 0;
}

int cmd_ps(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    DIR *proc;
    struct dirent *entry;
    proc_info_t *procs = NULL;
    size_t nprocs = 0;
    size_t capacity = 256;

    proc = opendir("/proc");
    if (!proc) {
        return proto_send_error(conn, id, strerror(errno));
    }

    procs = malloc(capacity * sizeof(proc_info_t));
    if (!procs) {
        closedir(proc);
        return proto_send_error(conn, id, "out of memory");
    }

    /* Enumerate /proc for PIDs */
    while ((entry = readdir(proc)) != NULL) {
        /* Skip non-numeric entries */
        if (!isdigit((unsigned char)entry->d_name[0])) {
            continue;
        }

        int pid = atoi(entry->d_name);
        if (pid <= 0) {
            continue;
        }

        /* Grow array if needed */
        if (nprocs >= capacity) {
            capacity *= 2;
            proc_info_t *tmp = realloc(procs, capacity * sizeof(proc_info_t));
            if (!tmp) {
                free(procs);
                closedir(proc);
                return proto_send_error(conn, id, "out of memory");
            }
            procs = tmp;
        }

        /* Read process info */
        if (read_proc_info(pid, &procs[nprocs]) == 0) {
            nprocs++;
        }
    }
    closedir(proc);

    LOG("ps: found %zu processes", nprocs);

    /* Build response: { "processes": [ { pid, ppid, name, state, cmdline }, ... ] } */
    resp_builder_t rb;
    if (rb_init(&rb, 8192) < 0) {
        free(procs);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "processes");

    /* Array of processes */
    rb_array(&rb, nprocs);

    for (size_t i = 0; i < nprocs; i++) {
        rb_map(&rb, 5);

        rb_str(&rb, "pid");
        rb_uint(&rb, (uint64_t)procs[i].pid);

        rb_str(&rb, "ppid");
        rb_uint(&rb, (uint64_t)procs[i].ppid);

        rb_str(&rb, "name");
        rb_str(&rb, procs[i].name);

        rb_str(&rb, "state");
        char state_str[2] = { procs[i].state, '\0' };
        rb_str(&rb, state_str);

        rb_str(&rb, "cmdline");
        rb_str(&rb, procs[i].cmdline);
    }

    free(procs);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
