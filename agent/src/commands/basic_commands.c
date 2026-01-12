/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Basic commands: ls, pwd, cd, cat
 * These are the core navigation and file reading commands.
 */

#define _POSIX_C_SOURCE 200809L
#define _XOPEN_SOURCE 500

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <sys/stat.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

/* =============================================================================
 * Command: ls
 *
 * List directory contents with file metadata.
 * ============================================================================= */

int cmd_ls(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    /* Parse path argument, default to cwd if not provided */
    char *arg_path = parse_string_arg(args, args_len, "path");
    char *resolved = NULL;
    const char *path;

    if (arg_path) {
        resolved = path_resolve(conn->cwd, arg_path);
        free(arg_path);
        if (!resolved) {
            return proto_send_error(conn, id, "out of memory");
        }
        path = resolved;
    } else {
        path = conn->cwd;
    }

    DIR *dir = opendir(path);
    if (!dir) {
        int err = errno;
        if (resolved) free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }

    /* First pass: count entries */
    size_t count = 0;
    struct dirent *ent;
    while ((ent = readdir(dir)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) {
            continue;
        }
        count++;
    }

    /* Build response */
    resp_builder_t rb;
    if (rb_init(&rb, 4096) < 0) {
        closedir(dir);
        if (resolved) free(resolved);
        return proto_send_error(conn, id, "out of memory");
    }

    /* { "entries": [ ... ] } */
    rb_map(&rb, 1);
    rb_str(&rb, "entries");
    rb_array(&rb, count);

    /* Second pass: build entries */
    rewinddir(dir);
    while ((ent = readdir(dir)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) {
            continue;
        }

        /* Build full path for stat */
        char fullpath[EDB_PATH_MAX];
        snprintf(fullpath, sizeof(fullpath), "%s/%s", path, ent->d_name);

        struct stat st;
        if (stat(fullpath, &st) < 0) {
            memset(&st, 0, sizeof(st));
        }

        /* Entry: { name, type, size, mode, mtime } */
        rb_map(&rb, 5);

        rb_str(&rb, "name");
        rb_str(&rb, ent->d_name);

        rb_str(&rb, "type");
        if (S_ISDIR(st.st_mode)) {
            rb_str(&rb, "dir");
        } else if (S_ISLNK(st.st_mode)) {
            rb_str(&rb, "link");
        } else if (S_ISREG(st.st_mode)) {
            rb_str(&rb, "file");
        } else {
            rb_str(&rb, "other");
        }

        rb_str(&rb, "size");
        rb_uint(&rb, (uint64_t)st.st_size);

        rb_str(&rb, "mode");
        rb_uint(&rb, (uint64_t)(st.st_mode & 0777));

        rb_str(&rb, "mtime");
        rb_uint(&rb, (uint64_t)st.st_mtime);
    }

    closedir(dir);
    if (resolved) free(resolved);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: pwd
 *
 * Print current working directory.
 * ============================================================================= */

int cmd_pwd(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    (void)args;
    (void)args_len;

    resp_builder_t rb;
    if (rb_init(&rb, 256) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "path");
    rb_str(&rb, conn->cwd);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: cd
 *
 * Change current working directory.
 * ============================================================================= */

int cmd_cd(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
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

    /* Check if path exists and is a directory */
    if (!path_exists(resolved)) {
        free(resolved);
        return proto_send_error(conn, id, "no such directory");
    }

    if (!path_is_dir(resolved)) {
        free(resolved);
        return proto_send_error(conn, id, "not a directory");
    }

    /* Get the real path (resolve symlinks, .., etc) */
    char realpath_buf[EDB_PATH_MAX];
    if (realpath(resolved, realpath_buf) == NULL) {
        free(resolved);
        return proto_send_error(conn, id, strerror(errno));
    }
    free(resolved);

    /* Update cwd */
    safe_strcpy(conn->cwd, realpath_buf, sizeof(conn->cwd));

    LOG("Changed directory to %s", conn->cwd);

    /* Build response with new path */
    resp_builder_t rb;
    if (rb_init(&rb, 256) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "path");
    rb_str(&rb, conn->cwd);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: realpath
 *
 * Resolve a path to its canonical absolute form.
 * Resolves symlinks, removes . and .. components.
 * ============================================================================= */

int cmd_realpath(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    /* Resolve relative to cwd */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    /* Get the canonical path */
    char realpath_buf[EDB_PATH_MAX];
    if (realpath(resolved, realpath_buf) == NULL) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }
    free(resolved);

    LOG("realpath: %s", realpath_buf);

    /* Build response */
    resp_builder_t rb;
    if (rb_init(&rb, 256) < 0) {
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "path");
    rb_str(&rb, realpath_buf);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}

/* =============================================================================
 * Command: cat
 *
 * Read and return file contents.
 * Handles both regular files and virtual files (e.g. /proc, /sys).
 * ============================================================================= */

int cmd_cat(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
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

    /* Open the file */
    FILE *f = fopen(resolved, "rb");
    if (!f) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }
    free(resolved);

    /*
     * Try to get file size via seeking. This works for regular files
     * but fails for virtual files (e.g. /proc, /sys) which have no size.
     */
    long file_size = -1;
    if (fseek(f, 0, SEEK_END) == 0) {
        file_size = ftell(f);
        if (file_size >= 0) {
            fseek(f, 0, SEEK_SET);
        }
    }

    uint8_t *content = NULL;
    size_t content_len = 0;

    if (file_size > 0) {
        /* Regular file with known size */
        if (file_size > EDB_MAX_MSG_SIZE - 1024) {
            fclose(f);
            return proto_send_error(conn, id, "file too large");
        }

        content = malloc((size_t)file_size);
        if (!content) {
            fclose(f);
            return proto_send_error(conn, id, "out of memory");
        }

        content_len = fread(content, 1, (size_t)file_size, f);
    } else {
        /*
         * Virtual file or empty file - read in chunks until EOF.
         * This handles /proc, /sys, and other special files.
         */
        size_t capacity = 4096;
        content = malloc(capacity);
        if (!content) {
            fclose(f);
            return proto_send_error(conn, id, "out of memory");
        }

        content_len = 0;
        size_t chunk;
        while ((chunk = fread(content + content_len, 1, capacity - content_len, f)) > 0) {
            content_len += chunk;

            /* Need more space? */
            if (content_len >= capacity) {
                if (capacity >= EDB_MAX_MSG_SIZE - 1024) {
                    free(content);
                    fclose(f);
                    return proto_send_error(conn, id, "file too large");
                }
                capacity *= 2;
                if (capacity > EDB_MAX_MSG_SIZE - 1024) {
                    capacity = EDB_MAX_MSG_SIZE - 1024;
                }
                uint8_t *new_content = realloc(content, capacity);
                if (!new_content) {
                    free(content);
                    fclose(f);
                    return proto_send_error(conn, id, "out of memory");
                }
                content = new_content;
            }
        }
    }
    fclose(f);

    /* Build response: { "content": <binary>, "size": <len> } */
    resp_builder_t rb;
    if (rb_init(&rb, content_len + 64) < 0) {
        free(content);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 2);

    rb_str(&rb, "content");
    rb_bin(&rb, content, content_len);
    free(content);

    rb_str(&rb, "size");
    rb_uint(&rb, (uint64_t)content_len);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
