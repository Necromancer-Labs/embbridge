/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Commands: ip_addr, ip_route - Network interface and routing info
 */

#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <dirent.h>
#include <errno.h>
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include "edb.h"
#include "commands.h"

/* =============================================================================
 * ip addr - Show network interfaces
 * ============================================================================= */

static int read_sysfs_string(const char *path, char *buf, size_t len)
{
    FILE *f = fopen(path, "r");
    if (!f) return -1;

    if (fgets(buf, len, f)) {
        /* Remove trailing newline */
        size_t l = strlen(buf);
        if (l > 0 && buf[l-1] == '\n') buf[l-1] = '\0';
    } else {
        buf[0] = '\0';
    }
    fclose(f);
    return 0;
}

static int get_iface_flags(const char *ifname)
{
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) return 0;

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    int flags = 0;
    if (ioctl(sock, SIOCGIFFLAGS, &ifr) == 0) {
        flags = ifr.ifr_flags;
    }
    close(sock);
    return flags;
}

static int get_iface_ipv4(const char *ifname, char *ip_buf, size_t len)
{
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) return -1;

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    if (ioctl(sock, SIOCGIFADDR, &ifr) == 0) {
        struct sockaddr_in *addr = (struct sockaddr_in *)&ifr.ifr_addr;
        inet_ntop(AF_INET, &addr->sin_addr, ip_buf, len);
        close(sock);
        return 0;
    }
    close(sock);
    return -1;
}

static int get_iface_netmask(const char *ifname, char *mask_buf, size_t len)
{
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) return -1;

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    if (ioctl(sock, SIOCGIFNETMASK, &ifr) == 0) {
        struct sockaddr_in *addr = (struct sockaddr_in *)&ifr.ifr_addr;
        inet_ntop(AF_INET, &addr->sin_addr, mask_buf, len);
        close(sock);
        return 0;
    }
    close(sock);
    return -1;
}

static int netmask_to_cidr(const char *netmask)
{
    struct in_addr addr;
    if (inet_pton(AF_INET, netmask, &addr) != 1) return 0;

    uint32_t n = ntohl(addr.s_addr);
    int cidr = 0;
    while (n & 0x80000000) {
        cidr++;
        n <<= 1;
    }
    return cidr;
}

static int get_iface_mtu(const char *ifname)
{
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) return 0;

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    int mtu = 0;
    if (ioctl(sock, SIOCGIFMTU, &ifr) == 0) {
        mtu = ifr.ifr_mtu;
    }
    close(sock);
    return mtu;
}

int cmd_ip_addr(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    DIR *dir = opendir("/sys/class/net");
    if (!dir) {
        return proto_send_error(conn, id, "cannot read network interfaces");
    }

    char output[8192];
    size_t offset = 0;

    struct dirent *ent;
    while ((ent = readdir(dir)) != NULL) {
        if (ent->d_name[0] == '.') continue;

        const char *ifname = ent->d_name;
        char path[256];
        char mac[32] = "";
        char operstate[16] = "";
        char ipv4[INET_ADDRSTRLEN] = "";
        char netmask[INET_ADDRSTRLEN] = "";

        /* Read MAC address */
        snprintf(path, sizeof(path), "/sys/class/net/%s/address", ifname);
        read_sysfs_string(path, mac, sizeof(mac));

        /* Read operational state */
        snprintf(path, sizeof(path), "/sys/class/net/%s/operstate", ifname);
        read_sysfs_string(path, operstate, sizeof(operstate));

        /* Get flags */
        int flags = get_iface_flags(ifname);

        /* Get IPv4 address */
        get_iface_ipv4(ifname, ipv4, sizeof(ipv4));
        get_iface_netmask(ifname, netmask, sizeof(netmask));

        /* Get MTU */
        int mtu = get_iface_mtu(ifname);

        /* Build flags string */
        char flagstr[128] = "";
        if (flags & IFF_UP) strcat(flagstr, "UP,");
        if (flags & IFF_BROADCAST) strcat(flagstr, "BROADCAST,");
        if (flags & IFF_LOOPBACK) strcat(flagstr, "LOOPBACK,");
        if (flags & IFF_RUNNING) strcat(flagstr, "RUNNING,");
        if (flags & IFF_MULTICAST) strcat(flagstr, "MULTICAST,");
        size_t flen = strlen(flagstr);
        if (flen > 0 && flagstr[flen-1] == ',') flagstr[flen-1] = '\0';

        /* Format output like ip addr */
        offset += snprintf(output + offset, sizeof(output) - offset,
            "%s: <%s> mtu %d state %s\n",
            ifname, flagstr, mtu, operstate);

        if (mac[0] && strcmp(mac, "00:00:00:00:00:00") != 0) {
            offset += snprintf(output + offset, sizeof(output) - offset,
                "    link/ether %s\n", mac);
        }

        if (ipv4[0]) {
            int cidr = netmask_to_cidr(netmask);
            offset += snprintf(output + offset, sizeof(output) - offset,
                "    inet %s/%d\n", ipv4, cidr);
        }

        if (offset >= sizeof(output) - 256) break;
    }

    closedir(dir);

    resp_builder_t rb;
    if (rb_init(&rb, offset + 64) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "content");
    rb_bin(&rb, (uint8_t *)output, offset);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * ip route - Show routing table
 * ============================================================================= */

int cmd_ip_route(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    FILE *f = fopen("/proc/net/route", "r");
    if (!f) {
        return proto_send_error(conn, id, "cannot read routing table");
    }

    char output[4096];
    size_t offset = 0;
    char line[256];

    /* Skip header */
    if (!fgets(line, sizeof(line), f)) {
        fclose(f);
        return proto_send_error(conn, id, "empty routing table");
    }

    while (fgets(line, sizeof(line), f)) {
        char iface[32];
        unsigned int dest, gateway, flags, mask;
        int refcnt, use, metric, mtu, window, irtt;

        if (sscanf(line, "%31s %X %X %X %d %d %d %X %d %d %d",
                   iface, &dest, &gateway, &flags, &refcnt, &use,
                   &metric, &mask, &mtu, &window, &irtt) < 8) {
            continue;
        }

        /* Skip down routes */
        if (!(flags & 0x0001)) continue;  /* RTF_UP */

        struct in_addr dest_addr, gw_addr, mask_addr;
        dest_addr.s_addr = dest;
        gw_addr.s_addr = gateway;
        mask_addr.s_addr = mask;

        char dest_str[INET_ADDRSTRLEN];
        char gw_str[INET_ADDRSTRLEN];

        inet_ntop(AF_INET, &dest_addr, dest_str, sizeof(dest_str));
        inet_ntop(AF_INET, &gw_addr, gw_str, sizeof(gw_str));

        int cidr = netmask_to_cidr(inet_ntoa(mask_addr));

        if (dest == 0) {
            /* Default route */
            offset += snprintf(output + offset, sizeof(output) - offset,
                "default via %s dev %s", gw_str, iface);
        } else {
            offset += snprintf(output + offset, sizeof(output) - offset,
                "%s/%d", dest_str, cidr);

            if (gateway != 0) {
                offset += snprintf(output + offset, sizeof(output) - offset,
                    " via %s", gw_str);
            }
            offset += snprintf(output + offset, sizeof(output) - offset,
                " dev %s", iface);
        }

        if (metric > 0) {
            offset += snprintf(output + offset, sizeof(output) - offset,
                " metric %d", metric);
        }

        offset += snprintf(output + offset, sizeof(output) - offset, "\n");

        if (offset >= sizeof(output) - 128) break;
    }

    fclose(f);

    if (offset == 0) {
        strcpy(output, "(no routes)\n");
        offset = strlen(output);
    }

    resp_builder_t rb;
    if (rb_init(&rb, offset + 64) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "content");
    rb_bin(&rb, (uint8_t *)output, offset);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
