/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Agent header file
 */

#ifndef EDB_H
#define EDB_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

/* =============================================================================
 * Constants
 * ============================================================================= */

#define EDB_VERSION         1
#define EDB_DEFAULT_PORT    1337
#define EDB_MAX_MSG_SIZE    (16 * 1024 * 1024)  /* 16 MB */
#define EDB_CHUNK_SIZE      (64 * 1024)         /* 64 KB */
#define EDB_PATH_MAX        4096
#define EDB_READBUF_SIZE    8192

/* =============================================================================
 * Message Types
 * ============================================================================= */

typedef enum {
    MSG_HELLO       = 1,
    MSG_HELLO_ACK   = 2,
    MSG_REQ         = 3,
    MSG_RESP        = 4,
    MSG_DATA        = 5,
} msg_type_t;

/* =============================================================================
 * Command Types
 * ============================================================================= */

typedef enum {
    CMD_UNKNOWN = 0,
    CMD_LS,
    CMD_CAT,
    CMD_PWD,
    CMD_CD,
    CMD_REALPATH,
    CMD_PULL,
    CMD_PUSH,
    CMD_EXEC,
    CMD_MKDIR,
    CMD_RM,
    CMD_MV,
    CMD_CP,
    CMD_CHMOD,
    CMD_TOUCH,
    CMD_UNAME,
    CMD_PS,
    CMD_NETSTAT,
    CMD_ENV,
    CMD_MTD,
    CMD_FIRMWARE,
    CMD_HEXDUMP,
    CMD_KILL_AGENT,
    CMD_REBOOT,
    CMD_WHOAMI,
    CMD_DMESG,
    CMD_STRINGS,
    CMD_CPUINFO,
    CMD_IP_ADDR,
    CMD_IP_ROUTE,
} cmd_type_t;

/* =============================================================================
 * Configuration
 * ============================================================================= */

typedef enum {
    MODE_CONNECT,   /* -c: Connect to client (reverse) */
    MODE_LISTEN,    /* -l: Listen for client (bind) */
} conn_mode_t;

typedef struct {
    conn_mode_t mode;
    char        host[256];
    uint16_t    port;
} config_t;

/* =============================================================================
 * Connection State
 * ============================================================================= */

typedef struct {
    int         sockfd;
    char        cwd[EDB_PATH_MAX];
    uint8_t     *recvbuf;
    size_t      recvbuf_size;
} conn_t;

/* =============================================================================
 * Protocol Functions (protocol.c)
 * ============================================================================= */

/*
 * Send a message over the connection.
 * Returns 0 on success, -1 on error.
 */
int proto_send(conn_t *conn, const uint8_t *data, size_t len);

/*
 * Receive a message from the connection.
 * Allocates buffer, caller must free.
 * Returns message length on success, -1 on error.
 */
int proto_recv(conn_t *conn, uint8_t **data, size_t *len);

/*
 * Send hello message.
 */
int proto_send_hello(conn_t *conn);

/*
 * Receive and validate hello message.
 */
int proto_recv_hello(conn_t *conn);

/*
 * Send hello_ack message.
 */
int proto_send_hello_ack(conn_t *conn);

/*
 * Send a response message.
 */
int proto_send_response(conn_t *conn, uint32_t id, bool ok,
                        const uint8_t *data, size_t data_len,
                        const char *error);

/*
 * Send an error response.
 */
int proto_send_error(conn_t *conn, uint32_t id, const char *error);

/*
 * Send a data chunk for file transfer.
 */
int proto_send_data(conn_t *conn, uint32_t id, uint32_t seq,
                    const uint8_t *data, size_t len, bool done);

/* =============================================================================
 * Transport Functions (transport.c)
 * ============================================================================= */

/*
 * Connect to a remote host (reverse mode).
 * Returns socket fd on success, -1 on error.
 */
int transport_connect(const char *host, uint16_t port);

/*
 * Create a listening socket on a port (bind mode).
 * Returns listening socket fd on success, -1 on error.
 */
int transport_listen(uint16_t port);

/*
 * Accept a connection on a listening socket.
 * Returns client socket fd on success, -1 on error.
 */
int transport_accept(int listenfd);

/*
 * Close the connection.
 */
void transport_close(int sockfd);

/*
 * Send raw bytes.
 */
int transport_send(int sockfd, const uint8_t *data, size_t len);

/*
 * Receive raw bytes.
 */
int transport_recv(int sockfd, uint8_t *data, size_t len);

/* =============================================================================
 * Command Handlers (src/commands/)
 * ============================================================================= */

/*
 * Parse command name to type. (cmd_dispatch.c)
 */
cmd_type_t cmd_parse(const char *name);

/*
 * Handle an incoming command. (cmd_dispatch.c)
 */
int cmd_handle(conn_t *conn, uint32_t id, cmd_type_t cmd,
               const uint8_t *args, size_t args_len);

/* Basic commands (basic_commands.c) */
int cmd_ls(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_cat(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_pwd(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_cd(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_realpath(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);

/* File transfer (file_transfer.c) */
int cmd_pull(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_push(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);

/* File operations (file_operations.c) */
int cmd_rm(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_mv(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_cp(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_mkdir(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_chmod(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_touch(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);

/* System commands (system_commands.c) */
int cmd_uname(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_ps(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_exec(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_netstat(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_kill_agent(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_reboot(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_whoami(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_dmesg(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_strings(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_cpuinfo(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_mtd(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_ip_addr(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);
int cmd_ip_route(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len);

/* =============================================================================
 * Path Utilities (src/commands/helpers.c)
 * ============================================================================= */

/*
 * Resolve a path relative to cwd.
 * Returns allocated string, caller must free.
 */
char *path_resolve(const char *cwd, const char *path);

/*
 * Check if path is a directory.
 */
bool path_is_dir(const char *path);

/*
 * Check if path exists.
 */
bool path_exists(const char *path);

/* =============================================================================
 * Utility Functions
 * ============================================================================= */

/*
 * Safe string copy.
 */
void safe_strcpy(char *dst, const char *src, size_t size);

/*
 * Log a message (debug builds only).
 */
#ifdef DEBUG
    #define LOG(fmt, ...) fprintf(stderr, "[edb] " fmt "\n", ##__VA_ARGS__)
#else
    #define LOG(fmt, ...) do {} while(0)
#endif

#endif /* EDB_H */
