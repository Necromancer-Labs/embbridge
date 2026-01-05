/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File operation commands: rm, mv, cp, mkdir, chmod, touch
 * These modify files and directories on the device.
 */

#define _POSIX_C_SOURCE 200809L

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* =============================================================================
 * Command: rm
 *
 * Remove a file or empty directory.
 * Uses unlink() for files, rmdir() for directories.
 * ============================================================================= */

int cmd_rm(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Check what we're deleting */
    struct stat st;
    if (stat(resolved, &st) < 0) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    int ret;
    if (S_ISDIR(st.st_mode)) {
        /* Directory - use rmdir (only works if empty) */
        ret = rmdir(resolved);
    } else {
        /* File or other - use unlink */
        ret = unlink(resolved);
    }

    if (ret < 0) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    LOG("rm: removed %s", resolved);
    free(resolved);

    /* Send empty success response */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }
    rb_map(&rb, 0);

    ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: mv
 *
 * Move or rename a file/directory.
 * Uses rename() syscall - works across directories on same filesystem.
 * ============================================================================= */

int cmd_mv(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *src = parse_string_arg(args, args_len, "src");
    if (!src) {
        return proto_send_error(conn, id, "missing src argument");
    }

    char *dst = parse_string_arg(args, args_len, "dst");
    if (!dst) {
        free(src);
        return proto_send_error(conn, id, "missing dst argument");
    }

    /* Resolve both paths */
    char *resolved_src = path_resolve(conn->cwd, src);
    free(src);
    if (!resolved_src) {
        free(dst);
        return proto_send_error(conn, id, "out of memory");
    }

    char *resolved_dst = path_resolve(conn->cwd, dst);
    free(dst);
    if (!resolved_dst) {
        free(resolved_src);
        return proto_send_error(conn, id, "out of memory");
    }

    /* Check source exists */
    if (!path_exists(resolved_src)) {
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, "source does not exist");
    }

    /* Perform the rename */
    if (rename(resolved_src, resolved_dst) < 0) {
        int err = errno;
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, strerror(err));
    }

    LOG("mv: %s -> %s", resolved_src, resolved_dst);
    free(resolved_src);
    free(resolved_dst);

    /* Send empty success response */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }
    rb_map(&rb, 0);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: mkdir
 *
 * Create a directory with optional mode.
 * Default mode is 0755 if not specified.
 * ============================================================================= */

int cmd_mkdir(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    /* Get optional mode, default to 0755 */
    uint64_t mode = 0755;
    parse_uint_arg(args, args_len, "mode", &mode);

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Create the directory */
    if (mkdir(resolved, (mode_t)mode) < 0) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    LOG("mkdir: created %s (mode %o)", resolved, (unsigned int)mode);
    free(resolved);

    /* Send empty success response */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }
    rb_map(&rb, 0);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: chmod
 *
 * Change file/directory permissions.
 * Expects mode as an integer (e.g., 0755, 0644).
 * ============================================================================= */

int cmd_chmod(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    uint64_t mode = 0;
    if (parse_uint_arg(args, args_len, "mode", &mode) < 0) {
        free(arg_path);
        return proto_send_error(conn, id, "missing mode argument");
    }

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Change permissions */
    if (chmod(resolved, (mode_t)mode) < 0) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    LOG("chmod: %s -> %o", resolved, (unsigned int)mode);
    free(resolved);

    /* Send empty success response */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }
    rb_map(&rb, 0);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: cp
 *
 * Copy a file from src to dst.
 * Preserves file permissions from the source.
 * Uses chunked reading/writing for memory efficiency.
 * ============================================================================= */

int cmd_cp(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *src = parse_string_arg(args, args_len, "src");
    if (!src) {
        return proto_send_error(conn, id, "missing src argument");
    }

    char *dst = parse_string_arg(args, args_len, "dst");
    if (!dst) {
        free(src);
        return proto_send_error(conn, id, "missing dst argument");
    }

    /* Resolve both paths */
    char *resolved_src = path_resolve(conn->cwd, src);
    free(src);
    if (!resolved_src) {
        free(dst);
        return proto_send_error(conn, id, "out of memory");
    }

    char *resolved_dst = path_resolve(conn->cwd, dst);
    free(dst);
    if (!resolved_dst) {
        free(resolved_src);
        return proto_send_error(conn, id, "out of memory");
    }

    /* Open source file */
    FILE *fsrc = fopen(resolved_src, "rb");
    if (!fsrc) {
        int err = errno;
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, strerror(err));
    }

    /* Get source file info */
    struct stat st;
    if (fstat(fileno(fsrc), &st) < 0) {
        int err = errno;
        fclose(fsrc);
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, strerror(err));
    }

    if (S_ISDIR(st.st_mode)) {
        fclose(fsrc);
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, "source is a directory");
    }

    /* Open destination file */
    FILE *fdst = fopen(resolved_dst, "wb");
    if (!fdst) {
        int err = errno;
        fclose(fsrc);
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, strerror(err));
    }

    /* Copy in chunks */
    uint8_t buf[8192];
    size_t total = 0;
    size_t n;

    while ((n = fread(buf, 1, sizeof(buf), fsrc)) > 0) {
        if (fwrite(buf, 1, n, fdst) != n) {
            int err = errno;
            fclose(fsrc);
            fclose(fdst);
            unlink(resolved_dst);  /* Clean up partial file */
            free(resolved_src);
            free(resolved_dst);
            return proto_send_error(conn, id, strerror(err));
        }
        total += n;
    }

    if (ferror(fsrc)) {
        fclose(fsrc);
        fclose(fdst);
        unlink(resolved_dst);
        free(resolved_src);
        free(resolved_dst);
        return proto_send_error(conn, id, "read error");
    }

    fclose(fsrc);
    fclose(fdst);

    /* Preserve source permissions */
    chmod(resolved_dst, st.st_mode & 0777);

    LOG("cp: %s -> %s (%zu bytes)", resolved_src, resolved_dst, total);
    free(resolved_src);
    free(resolved_dst);

    /* Send empty success response */
    resp_builder_t rb;
    if (rb_init(&rb, 32) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }
    rb_map(&rb, 0);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: touch
 *
 * Create an empty file or update timestamps.
 * ============================================================================= */

int cmd_touch(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;
    return proto_send_error(conn, id, "not implemented");
}
