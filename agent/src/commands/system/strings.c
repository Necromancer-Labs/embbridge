/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Command: strings - Extract printable strings from a file
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

#include "edb.h"
#include "commands.h"

int cmd_strings(conn_t *conn, uint32_t id, const uint8_t *args, size_t args_len)
{
    char *arg_path = parse_string_arg(args, args_len, "path");
    if (!arg_path) {
        return proto_send_error(conn, id, "missing path argument");
    }

    /* Get optional min_len parameter (default: 4) */
    uint64_t min_len = 4;
    parse_uint_arg(args, args_len, "min_len", &min_len);

    /* Resolve the path */
    char *resolved = path_resolve(conn->cwd, arg_path);
    free(arg_path);

    if (!resolved) {
        return proto_send_error(conn, id, "out of memory");
    }

    FILE *f = fopen(resolved, "rb");
    if (!f) {
        int err = errno;
        free(resolved);
        return proto_send_error(conn, id, strerror(err));
    }
    free(resolved);

    /* Read file and extract strings */
    size_t capacity = 4096;
    size_t output_len = 0;
    char *output = malloc(capacity);
    if (!output) {
        fclose(f);
        return proto_send_error(conn, id, "out of memory");
    }

    char current_string[1024];
    size_t current_len = 0;
    int c;

    while ((c = fgetc(f)) != EOF) {
        /* Check if printable ASCII (32-126) or tab/newline */
        if ((c >= 32 && c <= 126) || c == '\t') {
            if (current_len < sizeof(current_string) - 1) {
                current_string[current_len++] = (char)c;
            }
        } else {
            /* End of string - check if long enough */
            if (current_len >= (size_t)min_len) {
                current_string[current_len] = '\0';

                /* Grow output buffer if needed */
                size_t needed = output_len + current_len + 2;
                if (needed > capacity) {
                    while (needed > capacity) capacity *= 2;
                    char *new_output = realloc(output, capacity);
                    if (!new_output) {
                        free(output);
                        fclose(f);
                        return proto_send_error(conn, id, "out of memory");
                    }
                    output = new_output;
                }

                /* Append string with newline */
                memcpy(output + output_len, current_string, current_len);
                output_len += current_len;
                output[output_len++] = '\n';
            }
            current_len = 0;
        }
    }

    /* Handle last string */
    if (current_len >= (size_t)min_len) {
        current_string[current_len] = '\0';
        size_t needed = output_len + current_len + 2;
        if (needed > capacity) {
            char *new_output = realloc(output, needed);
            if (!new_output) {
                free(output);
                fclose(f);
                return proto_send_error(conn, id, "out of memory");
            }
            output = new_output;
        }
        memcpy(output + output_len, current_string, current_len);
        output_len += current_len;
        output[output_len++] = '\n';
    }

    fclose(f);

    LOG("strings: extracted %zu bytes of strings", output_len);

    resp_builder_t rb;
    if (rb_init(&rb, output_len + 64) < 0) {
        free(output);
        return proto_send_error(conn, id, "out of memory");
    }

    rb_map(&rb, 1);
    rb_str(&rb, "content");
    rb_bin(&rb, (uint8_t *)output, output_len);

    free(output);

    int ret = proto_send_response(conn, id, true, rb.buf, rb.len, NULL);
    rb_free(&rb);
    return ret;
}
