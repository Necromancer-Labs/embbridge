/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * TCP transport layer
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <fcntl.h>

#include "edb.h"

/* -----------------------------------------------------------------------------
 * Connect to Remote Host (Reverse Mode)
 * ----------------------------------------------------------------------------- */

int transport_connect(const char *host, uint16_t port)
{
    struct addrinfo hints, *res, *rp;
    char port_str[8];
    int sockfd = -1;
    int ret;

    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;      /* Allow IPv4 or IPv6 */
    hints.ai_socktype = SOCK_STREAM;  /* TCP */

    snprintf(port_str, sizeof(port_str), "%d", port);

    ret = getaddrinfo(host, port_str, &hints, &res);
    if (ret != 0) {
        LOG("getaddrinfo failed: %s", gai_strerror(ret));
        return -1;
    }

    /* Try each address until we successfully connect */
    for (rp = res; rp != NULL; rp = rp->ai_next) {
        sockfd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (sockfd < 0) {
            continue;
        }

        if (connect(sockfd, rp->ai_addr, rp->ai_addrlen) == 0) {
            break;  /* Success */
        }

        close(sockfd);
        sockfd = -1;
    }

    freeaddrinfo(res);

    if (sockfd < 0) {
        LOG("Failed to connect to %s:%d", host, port);
        return -1;
    }

    /* Set TCP_NODELAY for lower latency */
    int flag = 1;
    setsockopt(sockfd, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));

    return sockfd;
}

/* -----------------------------------------------------------------------------
 * Listen for Connection (Bind Mode)
 * ----------------------------------------------------------------------------- */

int transport_listen(uint16_t port)
{
    struct sockaddr_in6 addr;
    int listenfd;
    int opt = 1;

    /* Create socket */
    listenfd = socket(AF_INET6, SOCK_STREAM, 0);
    if (listenfd < 0) {
        /* Fall back to IPv4 */
        listenfd = socket(AF_INET, SOCK_STREAM, 0);
        if (listenfd < 0) {
            LOG("socket failed: %s", strerror(errno));
            return -1;
        }

        struct sockaddr_in addr4;
        memset(&addr4, 0, sizeof(addr4));
        addr4.sin_family = AF_INET;
        addr4.sin_port = htons(port);
        addr4.sin_addr.s_addr = INADDR_ANY;

        setsockopt(listenfd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

        if (bind(listenfd, (struct sockaddr *)&addr4, sizeof(addr4)) < 0) {
            LOG("bind failed: %s", strerror(errno));
            close(listenfd);
            return -1;
        }
    } else {
        /* IPv6 socket, also accept IPv4 */
        int no = 0;
        setsockopt(listenfd, IPPROTO_IPV6, IPV6_V6ONLY, &no, sizeof(no));
        setsockopt(listenfd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

        memset(&addr, 0, sizeof(addr));
        addr.sin6_family = AF_INET6;
        addr.sin6_port = htons(port);
        addr.sin6_addr = in6addr_any;

        if (bind(listenfd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
            LOG("bind failed: %s", strerror(errno));
            close(listenfd);
            return -1;
        }
    }

    /* Allow multiple pending connections for fork-on-accept */
    if (listen(listenfd, 5) < 0) {
        LOG("listen failed: %s", strerror(errno));
        close(listenfd);
        return -1;
    }

    LOG("Listening on port %d", port);

    return listenfd;
}

/* -----------------------------------------------------------------------------
 * Accept a Connection
 * ----------------------------------------------------------------------------- */

int transport_accept(int listenfd)
{
    int connfd;

    LOG("Waiting for connection...");

    connfd = accept(listenfd, NULL, NULL);
    if (connfd < 0) {
        LOG("accept failed: %s", strerror(errno));
        return -1;
    }

    /* Set TCP_NODELAY for lower latency */
    int flag = 1;
    setsockopt(connfd, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));

    LOG("Client connected");

    return connfd;
}

/* -----------------------------------------------------------------------------
 * Close Connection
 * ----------------------------------------------------------------------------- */

void transport_close(int sockfd)
{
    if (sockfd >= 0) {
        shutdown(sockfd, SHUT_RDWR);
        close(sockfd);
    }
}

/* -----------------------------------------------------------------------------
 * Send Raw Bytes
 * ----------------------------------------------------------------------------- */

int transport_send(int sockfd, const uint8_t *data, size_t len)
{
    size_t sent = 0;

    while (sent < len) {
        ssize_t n = send(sockfd, data + sent, len - sent, 0);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            LOG("send failed: %s", strerror(errno));
            return -1;
        }
        if (n == 0) {
            LOG("connection closed");
            return -1;
        }
        sent += n;
    }

    return 0;
}

/* -----------------------------------------------------------------------------
 * Receive Raw Bytes
 * ----------------------------------------------------------------------------- */

int transport_recv(int sockfd, uint8_t *data, size_t len)
{
    size_t received = 0;

    while (received < len) {
        ssize_t n = recv(sockfd, data + received, len - received, 0);
        if (n < 0) {
            if (errno == EINTR) {
                continue;
            }
            LOG("recv failed: %s", strerror(errno));
            return -1;
        }
        if (n == 0) {
            LOG("connection closed");
            return -1;
        }
        received += n;
    }

    return 0;
}
