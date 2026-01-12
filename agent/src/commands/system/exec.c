/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: exec - Execute a something directly (execv)
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* Read all data from a file descriptor into a dynamically allocated buffer */
static char *read_fd_all(int fd, size_t *out_len)
{
    char *buf = NULL;
    size_t cap = 0;
    size_t len = 0;
    char tmp[4096];
    ssize_t n;

    while ((n = read(fd, tmp, sizeof(tmp))) > 0) {
        if (len + (size_t)n > cap) {
            size_t newcap = cap ? cap * 2 : 4096;
            if (newcap < len + (size_t)n) {
                newcap = len + (size_t)n;
            }
            char *newbuf = realloc(buf, newcap);
            if (!newbuf) {
                free(buf);
                return NULL;
            }
            buf = newbuf;
            cap = newcap;
        }
        memcpy(buf + len, tmp, (size_t)n);
        len += (size_t)n;
    }

    *out_len = len;
    return buf ? buf : calloc(1, 1);  /* Return empty string if nothing read */
}

/* Parse command string into argv array (space-separated) */
static char **parse_argv(char *cmd, int *argc_out)
{
    int argc = 0;
    int cap = 16;
    char **argv = malloc((size_t)cap * sizeof(char *));
    if (!argv) return NULL;

    char *p = cmd;
    while (*p) {
        /* Skip whitespace */
        while (*p && (*p == ' ' || *p == '\t')) p++;
        if (!*p) break;

        /* Start of token */
        char *start = p;

        /* Find end of token */
        while (*p && *p != ' ' && *p != '\t') p++;

        /* Null-terminate this token */
        if (*p) {
            *p = '\0';
            p++;
        }

        /* Grow argv if needed */
        if (argc >= cap - 1) {
            cap *= 2;
            char **newargv = realloc(argv, (size_t)cap * sizeof(char *));
            if (!newargv) {
                free(argv);
                return NULL;
            }
            argv = newargv;
        }

        argv[argc++] = start;
    }

    argv[argc] = NULL;  /* NULL-terminate argv */
    *argc_out = argc;
    return argv;
}

int cmd_exec(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *command = parse_string_arg(args, args_len, "command");
    if (!command) {
        return proto_send_error(conn, id, "missing command argument");
    }

    LOG("exec: running '%s'", command);

    /* Parse command into argv */
    int argc;
    char **argv = parse_argv(command, &argc);
    if (!argv || argc == 0) {
        free(command);
        free(argv);
        return proto_send_error(conn, id, "invalid command");
    }

    /* argv[0] is the executable path */
    const char *executable = argv[0];

    /* Create pipes for stdout and stderr */
    int stdout_pipe[2], stderr_pipe[2];
    if (pipe(stdout_pipe) < 0) {
        int err = errno;
        free(command);
        free(argv);
        return proto_send_error(conn, id, strerror(err));
    }
    if (pipe(stderr_pipe) < 0) {
        int err = errno;
        close(stdout_pipe[0]);
        close(stdout_pipe[1]);
        free(command);
        free(argv);
        return proto_send_error(conn, id, strerror(err));
    }

    pid_t pid = fork();
    if (pid < 0) {
        int err = errno;
        close(stdout_pipe[0]);
        close(stdout_pipe[1]);
        close(stderr_pipe[0]);
        close(stderr_pipe[1]);
        free(command);
        free(argv);
        return proto_send_error(conn, id, strerror(err));
    }

    if (pid == 0) {
        /* Child process */
        close(stdout_pipe[0]);  /* Close read ends */
        close(stderr_pipe[0]);

        /* Redirect stdout and stderr to pipes */
        dup2(stdout_pipe[1], STDOUT_FILENO);
        dup2(stderr_pipe[1], STDERR_FILENO);

        close(stdout_pipe[1]);
        close(stderr_pipe[1]);

        /* Execute directly - no shell */
        execv(executable, argv);

        /* If exec fails, write error and exit */
        /* Note: can't use fprintf as it may not flush before _exit */
        const char *err = strerror(errno);
        write(STDERR_FILENO, "exec: ", 6);
        write(STDERR_FILENO, err, strlen(err));
        write(STDERR_FILENO, "\n", 1);
        _exit(127);
    }

    /* Parent process */
    close(stdout_pipe[1]);  /* Close write ends */
    close(stderr_pipe[1]);

    /* Read stdout and stderr */
    size_t stdout_len = 0, stderr_len = 0;
    char *stdout_buf = read_fd_all(stdout_pipe[0], &stdout_len);
    char *stderr_buf = read_fd_all(stderr_pipe[0], &stderr_len);

    close(stdout_pipe[0]);
    close(stderr_pipe[0]);

    /* Wait for child to finish */
    int status = 0;
    waitpid(pid, &status, 0);

    int exit_code = 0;
    if (WIFEXITED(status)) {
        exit_code = WEXITSTATUS(status);
    } else if (WIFSIGNALED(status)) {
        exit_code = 128 + WTERMSIG(status);
    }

    free(command);
    free(argv);

    if (!stdout_buf || !stderr_buf) {
        free(stdout_buf);
        free(stderr_buf);
        return proto_send_error(conn, id, "out of memory");
    }

    LOG("exec: exit_code=%d, stdout=%zu bytes, stderr=%zu bytes",
        exit_code, stdout_len, stderr_len);

    /* Build response: { stdout: bytes, stderr: bytes, exit_code: int } */
    resp_builder_t rb;
    if (rb_init(&rb, 256 + stdout_len + stderr_len) < 0) {
        free(stdout_buf);
        free(stderr_buf);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 3);

    rb_str(&rb, "stdout");
    rb_bin(&rb, (uint8_t *)stdout_buf, stdout_len);

    rb_str(&rb, "stderr");
    rb_bin(&rb, (uint8_t *)stderr_buf, stderr_len);

    rb_str(&rb, "exit_code");
    rb_uint(&rb, (uint64_t)exit_code);

    free(stdout_buf);
    free(stderr_buf);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
