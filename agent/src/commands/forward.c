/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Port forwarding: tunnel TCP connections through the agent
 *
 * This allows the client to access services on the device's network,
 * such as RTSP streams, web UIs, or pivoting to other hosts.
 *
 * Architecture:
 *   Client <--embbridge--> Agent <--TCP/UDP--> Target
 *
 * The tunnel is single-stream for simplicity:
 *   - One active tunnel at a time per connection
 *   - Tunnel data flows via MSG_TUNNEL_DATA messages
 *   - Closing the tunnel from either side tears it down
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <netdb.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <sys/select.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include "edb.h"
#include "commands.h"

/* =============================================================================
 * Constants
 * ============================================================================= */

#define CONNECT_TIMEOUT  10     /* Seconds to wait for target connection */

/* =============================================================================
 * Helper: Connect to target host:port
 * ============================================================================= */

/*
 * Connects to the specified host and port.
 * Returns socket fd on success, -1 on error.
 */
static int connect_to_target(const char *host, uint16_t port, int socktype)
{
    struct addrinfo hints, *res, *rp;
    char port_str[16];
    int sockfd = -1;
    int ret;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;     /* Allow IPv4 or IPv6 */
    hints.ai_socktype = socktype;

    snprintf(port_str, sizeof(port_str), "%u", port);

    ret = getaddrinfo(host, port_str, &hints, &res);
    if (ret != 0) {
        LOG("getaddrinfo failed: %s", gai_strerror(ret));
        return -1;
    }

    /* Try each address until we connect */
    for (rp = res; rp != NULL; rp = rp->ai_next) {
        sockfd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (sockfd < 0) {
            continue;
        }

        /* Set socket to non-blocking for connect timeout */
        int flags = fcntl(sockfd, F_GETFL, 0);
        fcntl(sockfd, F_SETFL, flags | O_NONBLOCK);

        ret = connect(sockfd, rp->ai_addr, rp->ai_addrlen);
        if (ret == 0) {
            /* Connected immediately */
            fcntl(sockfd, F_SETFL, flags);  /* Restore blocking mode */
            break;
        }

        if (errno == EINPROGRESS) {
            /* Wait for connection with timeout */
            fd_set wfds;
            struct timeval tv;

            FD_ZERO(&wfds);
            FD_SET(sockfd, &wfds);
            tv.tv_sec = CONNECT_TIMEOUT;
            tv.tv_usec = 0;

            ret = select(sockfd + 1, NULL, &wfds, NULL, &tv);
            if (ret > 0) {
                /* Check if connection succeeded */
                int err = 0;
                socklen_t errlen = sizeof(err);
                getsockopt(sockfd, SOL_SOCKET, SO_ERROR, &err, &errlen);
                if (err == 0) {
                    fcntl(sockfd, F_SETFL, flags);  /* Restore blocking mode */
                    break;
                }
            }
        }

        close(sockfd);
        sockfd = -1;
    }

    freeaddrinfo(res);
    return sockfd;
}

/* =============================================================================
 * CMD_FORWARD_OPEN: Open a tunnel to target host:port
 * ============================================================================= */

/*
 * Opens a port forward tunnel.
 *
 * Args (msgpack map):
 *   - host: string  - Target hostname or IP
 *   - port: uint16  - Target port
 *   - proto: string - "tcp" (default) or "udp"
 *
 * Response:
 *   - ok: true on success, false on failure
 *   - error: string (only if ok=false)
 */
int cmd_forward_open(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *host = NULL;
    uint64_t port_val = 0;
    uint16_t port;
    int sockfd;

    /* Close any existing tunnel first */
    if (conn->tunnel_fd >= 0) {
        close(conn->tunnel_fd);
        conn->tunnel_fd = -1;
        conn->tunnel_id = 0;
        conn->tunnel_host[0] = '\0';
        conn->tunnel_port = 0;
        LOG("Closed existing tunnel");
    }

    /* Parse arguments */
    host = parse_string_arg(args, args_len, "host");
    if (!host || host[0] == '\0') {
        if (host) free(host);
        return proto_send_error(conn, id, "host is required");
    }

    if (parse_uint_arg(args, args_len, "port", &port_val) < 0 || port_val == 0) {
        free(host);
        return proto_send_error(conn, id, "port is required");
    }
    port = (uint16_t)port_val;

    /* Parse optional protocol (default: tcp) */
    int socktype = SOCK_STREAM;
    char *proto = parse_string_arg(args, args_len, "proto");
    if (proto) {
        if (strcmp(proto, "udp") == 0) {
            socktype = SOCK_DGRAM;
        } else if (strcmp(proto, "tcp") != 0) {
            free(proto);
            free(host);
            return proto_send_error(conn, id, "proto must be 'tcp' or 'udp'");
        }
        free(proto);
    }

    LOG("Forward open: %s:%u proto=%s", host, port,
        socktype == SOCK_DGRAM ? "udp" : "tcp");

    /* Connect to target */
    sockfd = connect_to_target(host, port, socktype);
    if (sockfd < 0) {
        char err[128];
        snprintf(err, sizeof(err), "failed to connect to %s:%u", host, port);
        free(host);
        return proto_send_error(conn, id, err);
    }

    /* Set non-blocking for async I/O in main loop */
    int flags = fcntl(sockfd, F_GETFL, 0);
    fcntl(sockfd, F_SETFL, flags | O_NONBLOCK);

    /* Store tunnel state (including host/port for reconnection) */
    conn->tunnel_fd = sockfd;
    conn->tunnel_id = id;
    snprintf(conn->tunnel_host, sizeof(conn->tunnel_host), "%s", host);
    conn->tunnel_port = port;
    conn->tunnel_socktype = socktype;

    free(host);

    LOG("Tunnel opened: fd=%d, id=%u", sockfd, id);

    /* Send success response */
    return proto_send_response(conn, id, true, NULL, 0, NULL);
}

/* =============================================================================
 * CMD_FORWARD_CLOSE: Close the current tunnel
 * ============================================================================= */

/*
 * Closes the current port forward tunnel.
 *
 * Args: none
 *
 * Response:
 *   - ok: true on success
 */
int cmd_forward_close(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    tunnel_close(conn);

    LOG("Tunnel closed by client");
    return proto_send_response(conn, id, true, NULL, 0, NULL);
}

/* =============================================================================
 * Tunnel Data Handling
 * ============================================================================= */

/*
 * Sends data from the tunnel target back to the client.
 * Called from the main loop when data is available on tunnel_fd.
 *
 * Message format:
 *   { "type": "tunnel_data", "id": <tunnel_id>, "data": <binary> }
 */
int tunnel_send_data(conn_t *conn, const uint8_t *data, size_t len)
{
    resp_builder_t rb;
    int ret;

    if (conn->tunnel_fd < 0 || len == 0) {
        return 0;
    }

    if (rb_init(&rb, 64 + len) < 0) {
        return -1;
    }

    /* Build tunnel_data message: 3 fields */
    rb_map(&rb, 3);

    rb_str(&rb, "type");
    rb_str(&rb, "tunnel_data");

    rb_str(&rb, "id");
    rb_uint(&rb, conn->tunnel_id);

    rb_str(&rb, "data");
    rb_bin(&rb, data, len);

    ret = proto_send(conn, rb.buf, rb.len);
    rb_free(&rb);

    return ret;
}

/*
 * Reconnect to the tunnel target (for handling HTTP/1.0 closes)
 */
static int tunnel_reconnect(conn_t *conn)
{
    int sockfd;
    int flags;

    if (conn->tunnel_host[0] == '\0' || conn->tunnel_port == 0) {
        return -1;  /* No stored target to reconnect to */
    }

    LOG("Reconnecting to %s:%u", conn->tunnel_host, conn->tunnel_port);

    sockfd = connect_to_target(conn->tunnel_host, conn->tunnel_port,
                               conn->tunnel_socktype);
    if (sockfd < 0) {
        LOG("Reconnect failed");
        return -1;
    }

    /* Set non-blocking */
    flags = fcntl(sockfd, F_GETFL, 0);
    fcntl(sockfd, F_SETFL, flags | O_NONBLOCK);

    conn->tunnel_fd = sockfd;
    LOG("Reconnected: fd=%d", sockfd);

    return 0;
}

/*
 * Handles incoming tunnel data from the client.
 * Writes the data to the tunnel socket (target).
 *
 * This function is called from handle_request() when a tunnel_data
 * message type is received.
 */
int tunnel_handle_data(conn_t *conn, const uint8_t *msg, size_t msg_len)
{
    const uint8_t *data = NULL;
    size_t data_len = 0;

    /* If tunnel fd is closed but we have stored host/port, reconnect (TCP only) */
    if (conn->tunnel_fd < 0) {
        if (conn->tunnel_socktype != SOCK_DGRAM &&
            conn->tunnel_host[0] != '\0' && conn->tunnel_port != 0) {
            if (tunnel_reconnect(conn) < 0) {
                LOG("Failed to reconnect to tunnel target");
                return -1;
            }
        } else {
            LOG("Tunnel data received but no tunnel active");
            return -1;
        }
    }

    /*
     * Parse the tunnel_data message to extract the binary data.
     * We need to find the "data" key in the msgpack map.
     *
     * For simplicity, we scan for the pattern. The message format is:
     *   { "type": "tunnel_data", "id": <uint>, "data": <bin> }
     */

    /* Skip to find "data" key - simplified scan */
    const char *data_key = "data";
    size_t key_len = 4;

    for (size_t i = 0; i + key_len + 2 < msg_len; i++) {
        /* Look for fixstr "data" (0xa4 'd' 'a' 't' 'a') */
        if (msg[i] == (0xa0 | key_len) &&
            memcmp(&msg[i + 1], data_key, key_len) == 0) {
            /* Found the key, next is the bin value */
            size_t pos = i + 1 + key_len;

            /* Parse bin header */
            if (pos >= msg_len) break;

            uint8_t marker = msg[pos++];
            if (marker == 0xc4) {
                /* bin8 */
                if (pos >= msg_len) break;
                data_len = msg[pos++];
            } else if (marker == 0xc5) {
                /* bin16 */
                if (pos + 2 > msg_len) break;
                data_len = (msg[pos] << 8) | msg[pos + 1];
                pos += 2;
            } else if (marker == 0xc6) {
                /* bin32 */
                if (pos + 4 > msg_len) break;
                data_len = ((uint32_t)msg[pos] << 24) |
                           ((uint32_t)msg[pos + 1] << 16) |
                           ((uint32_t)msg[pos + 2] << 8) |
                           (uint32_t)msg[pos + 3];
                pos += 4;
            } else {
                LOG("Unknown bin marker: 0x%02x", marker);
                break;
            }

            if (pos + data_len > msg_len) {
                LOG("Truncated tunnel data");
                break;
            }

            data = &msg[pos];
            break;
        }
    }

    if (!data || data_len == 0) {
        LOG("Failed to parse tunnel data");
        return -1;
    }

    /* Write data to tunnel socket */
    if (conn->tunnel_socktype == SOCK_DGRAM) {
        /* UDP: single atomic send (datagrams are not partial) */
        ssize_t n = send(conn->tunnel_fd, data, data_len, 0);
        if (n < 0) {
            LOG("UDP tunnel send error: %s", strerror(errno));
            /* Don't close on UDP error — connectionless */
        }
    } else {
        /* TCP: partial write loop */
        size_t written = 0;
        while (written < data_len) {
            ssize_t n = write(conn->tunnel_fd, data + written, data_len - written);
            if (n < 0) {
                if (errno == EAGAIN || errno == EWOULDBLOCK) {
                    fd_set wfds;
                    FD_ZERO(&wfds);
                    FD_SET(conn->tunnel_fd, &wfds);
                    select(conn->tunnel_fd + 1, NULL, &wfds, NULL, NULL);
                    continue;
                }
                LOG("Tunnel write error: %s", strerror(errno));
                tunnel_close(conn);
                return -1;
            }
            written += n;
        }
    }

    return 0;
}

/*
 * Closes the current tunnel and cleans up state.
 * This fully closes the tunnel - clearing host/port so reconnection is not possible.
 */
void tunnel_close(conn_t *conn)
{
    if (conn->tunnel_fd >= 0) {
        close(conn->tunnel_fd);
        conn->tunnel_fd = -1;
    }
    conn->tunnel_id = 0;
    conn->tunnel_host[0] = '\0';
    conn->tunnel_port = 0;
    conn->tunnel_socktype = SOCK_STREAM;
    LOG("Tunnel closed");
}
