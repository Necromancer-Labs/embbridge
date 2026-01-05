/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Agent main entry point
 *
 * Usage:
 *   ./edb-agent -c <host:port>   Connect to client (reverse)
 *   ./edb-agent -l <port>        Listen for client (bind)
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <sys/wait.h>

#include "edb.h"

/* -----------------------------------------------------------------------------
 * Globals
 * ----------------------------------------------------------------------------- */

static volatile int g_running = 1;

/* -----------------------------------------------------------------------------
 * Signal Handling
 * ----------------------------------------------------------------------------- */

static void signal_handler(int sig)
{
    (void)sig;
    g_running = 0;
}

static void sigchld_handler(int sig)
{
    (void)sig;
    /* Reap all zombie children without blocking */
    while (waitpid(-1, NULL, WNOHANG) > 0);
}

static void setup_signals(void)
{
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = signal_handler;
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);

    /* Ignore SIGPIPE - handle errors on write instead */
    sa.sa_handler = SIG_IGN;
    sigaction(SIGPIPE, &sa, NULL);

    /* Handle SIGCHLD to reap zombie children (bind mode) */
    sa.sa_handler = sigchld_handler;
    sa.sa_flags = SA_RESTART;  /* Restart interrupted syscalls */
    sigaction(SIGCHLD, &sa, NULL);
}

/* -----------------------------------------------------------------------------
 * Usage
 * ----------------------------------------------------------------------------- */

static void print_usage(const char *prog)
{
    fprintf(stderr, "embbridge agent v%d\n\n", EDB_VERSION);
    fprintf(stderr, "Usage:\n");
    fprintf(stderr, "  %s -c <host:port>   Connect to client (reverse)\n", prog);
    fprintf(stderr, "  %s -l <port>        Listen for client (bind)\n", prog);
    fprintf(stderr, "\n");
    fprintf(stderr, "Examples:\n");
    fprintf(stderr, "  %s -c 192.168.1.100:1337\n", prog);
    fprintf(stderr, "  %s -l 1337\n", prog);
}

/* -----------------------------------------------------------------------------
 * Argument Parsing
 * ----------------------------------------------------------------------------- */

static int parse_host_port(const char *str, char *host, size_t host_size, uint16_t *port)
{
    const char *colon = strrchr(str, ':');
    if (!colon) {
        return -1;
    }

    size_t host_len = colon - str;
    if (host_len >= host_size) {
        return -1;
    }

    memcpy(host, str, host_len);
    host[host_len] = '\0';

    *port = (uint16_t)atoi(colon + 1);
    if (*port == 0) {
        return -1;
    }

    return 0;
}

static int parse_args(int argc, char *argv[], config_t *cfg)
{
    memset(cfg, 0, sizeof(*cfg));
    cfg->port = EDB_DEFAULT_PORT;

    if (argc < 3) {
        return -1;
    }

    if (strcmp(argv[1], "-c") == 0) {
        /* Connect mode (reverse) */
        cfg->mode = MODE_CONNECT;
        if (parse_host_port(argv[2], cfg->host, sizeof(cfg->host), &cfg->port) < 0) {
            fprintf(stderr, "Error: Invalid host:port format\n");
            return -1;
        }
    } else if (strcmp(argv[1], "-l") == 0) {
        /* Listen mode (bind) */
        cfg->mode = MODE_LISTEN;
        cfg->port = (uint16_t)atoi(argv[2]);
        if (cfg->port == 0) {
            fprintf(stderr, "Error: Invalid port\n");
            return -1;
        }
    } else {
        return -1;
    }

    return 0;
}

/* -----------------------------------------------------------------------------
 * Main Loop
 * ----------------------------------------------------------------------------- */

static int do_handshake(conn_t *conn, conn_mode_t mode)
{
    uint8_t *msg = NULL;
    size_t msg_len;

    if (mode == MODE_CONNECT) {
        /* Reverse mode: agent sends hello first */
        LOG("Sending hello...");
        if (proto_send_hello(conn) < 0) {
            LOG("Failed to send hello");
            return -1;
        }

        /* Wait for hello_ack */
        LOG("Waiting for hello_ack...");
        if (proto_recv(conn, &msg, &msg_len) < 0) {
            LOG("Failed to receive hello_ack");
            return -1;
        }
        /* TODO: Validate hello_ack */
        free(msg);

    } else {
        /* Bind mode: agent receives hello first */
        LOG("Waiting for hello...");
        if (proto_recv(conn, &msg, &msg_len) < 0) {
            LOG("Failed to receive hello");
            return -1;
        }
        /* TODO: Validate hello */
        free(msg);

        /* Send hello_ack */
        LOG("Sending hello_ack...");
        if (proto_send_hello_ack(conn) < 0) {
            LOG("Failed to send hello_ack");
            return -1;
        }
    }

    LOG("Handshake complete");
    return 0;
}

/* Forward declaration */
int handle_request(conn_t *conn, const uint8_t *msg, size_t msg_len);

static int run_session(conn_t *conn, conn_mode_t mode)
{
    uint8_t *msg = NULL;
    size_t msg_len;
    int ret;

    /* Perform handshake */
    if (do_handshake(conn, mode) < 0) {
        return -1;
    }

    LOG("Session started, cwd=%s", conn->cwd);

    while (g_running) {
        /* Receive a message */
        ret = proto_recv(conn, &msg, &msg_len);
        if (ret < 0) {
            LOG("Connection closed or error");
            break;
        }

        LOG("Received message (%zu bytes)", msg_len);

        /* Handle the request */
        if (handle_request(conn, msg, msg_len) < 0) {
            LOG("Failed to handle request");
        }

        free(msg);
        msg = NULL;
    }

    if (msg) {
        free(msg);
    }

    return 0;
}

/* -----------------------------------------------------------------------------
 * Initialize Connection State
 * ----------------------------------------------------------------------------- */

static int init_conn(conn_t *conn)
{
    memset(conn, 0, sizeof(*conn));
    conn->sockfd = -1;

    /* Set initial working directory */
    if (getcwd(conn->cwd, sizeof(conn->cwd)) == NULL) {
        safe_strcpy(conn->cwd, "/", sizeof(conn->cwd));
    }

    /* Allocate receive buffer */
    conn->recvbuf_size = EDB_READBUF_SIZE;
    conn->recvbuf = malloc(conn->recvbuf_size);
    if (!conn->recvbuf) {
        fprintf(stderr, "Error: Out of memory\n");
        return -1;
    }

    return 0;
}

static void cleanup_conn(conn_t *conn)
{
    if (conn->sockfd >= 0) {
        transport_close(conn->sockfd);
    }
    if (conn->recvbuf) {
        free(conn->recvbuf);
    }
}

/* -----------------------------------------------------------------------------
 * Run in Reverse Mode (agent connects to client)
 * ----------------------------------------------------------------------------- */

static int run_reverse_mode(config_t *cfg)
{
    conn_t conn;

    if (init_conn(&conn) < 0) {
        return 1;
    }

    LOG("Connecting to %s:%d", cfg->host, cfg->port);
    conn.sockfd = transport_connect(cfg->host, cfg->port);
    if (conn.sockfd < 0) {
        fprintf(stderr, "Error: Failed to connect to %s:%d\n", cfg->host, cfg->port);
        cleanup_conn(&conn);
        return 1;
    }
    LOG("Connected");

    run_session(&conn, MODE_CONNECT);

    cleanup_conn(&conn);
    return 0;
}

/* -----------------------------------------------------------------------------
 * Run in Bind Mode (agent listens, forks for each client)
 * ----------------------------------------------------------------------------- */

static int run_bind_mode(config_t *cfg)
{
    int listenfd;
    int connfd;
    pid_t pid;

    LOG("Starting bind mode on port %d", cfg->port);

    listenfd = transport_listen(cfg->port);
    if (listenfd < 0) {
        fprintf(stderr, "Error: Failed to listen on port %d\n", cfg->port);
        return 1;
    }

    /* Accept loop - parent keeps accepting, children handle sessions */
    while (g_running) {
        connfd = transport_accept(listenfd);
        if (connfd < 0) {
            if (!g_running) {
                break;  /* Clean shutdown */
            }
            LOG("Accept failed, continuing...");
            continue;
        }

        pid = fork();
        if (pid < 0) {
            LOG("fork failed");
            transport_close(connfd);
            continue;
        }

        if (pid == 0) {
            /* Child process: handle this client */
            conn_t conn;

            close(listenfd);  /* Child doesn't need listening socket */

            if (init_conn(&conn) < 0) {
                _exit(1);
            }
            conn.sockfd = connfd;

            LOG("Child %d handling client", getpid());
            run_session(&conn, MODE_LISTEN);

            cleanup_conn(&conn);
            _exit(0);
        } else {
            /* Parent process: continue accepting */
            close(connfd);  /* Parent doesn't need client socket */
            LOG("Forked child %d for client", pid);
        }
    }

    close(listenfd);
    return 0;
}

/* -----------------------------------------------------------------------------
 * Main
 * ----------------------------------------------------------------------------- */

int main(int argc, char *argv[])
{
    config_t cfg;

    /* Parse arguments */
    if (parse_args(argc, argv, &cfg) < 0) {
        print_usage(argv[0]);
        return 1;
    }

    /* Setup signal handlers */
    setup_signals();

    /* Run in appropriate mode */
    if (cfg.mode == MODE_CONNECT) {
        return run_reverse_mode(&cfg);
    } else {
        return run_bind_mode(&cfg);
    }
}
