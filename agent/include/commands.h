/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Internal header for command implementations.
 * Provides shared utilities: argument parsing, response building, path helpers.
 */

#ifndef COMMANDS_H
#define COMMANDS_H

#include "edb.h"
#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

/* =============================================================================
 * Response Builder
 *
 * A simple buffer builder for constructing MessagePack responses.
 * Grows automatically as needed.
 * ============================================================================= */

typedef struct {
    uint8_t *buf;
    size_t   cap;
    size_t   len;
} resp_builder_t;

/* Initialize a response builder with given capacity */
int rb_init(resp_builder_t *rb, size_t cap);

/* Free the response builder's buffer */
void rb_free(resp_builder_t *rb);

/* Write primitives */
int rb_u8(resp_builder_t *rb, uint8_t v);
int rb_u16be(resp_builder_t *rb, uint16_t v);
int rb_u32be(resp_builder_t *rb, uint32_t v);
int rb_raw(resp_builder_t *rb, const void *data, size_t len);

/* Write MessagePack types */
int rb_str(resp_builder_t *rb, const char *s);
int rb_bin(resp_builder_t *rb, const uint8_t *data, size_t len);
int rb_uint(resp_builder_t *rb, uint64_t v);
int rb_map(resp_builder_t *rb, size_t count);
int rb_array(resp_builder_t *rb, size_t count);

/* =============================================================================
 * Argument Parsing
 *
 * Parse values from MessagePack-encoded argument maps.
 * These handle the binary format manually to avoid external dependencies.
 * ============================================================================= */

/*
 * Parse a string value from a MessagePack map.
 * Returns allocated string (caller must free), or NULL if not found.
 */
char *parse_string_arg(const uint8_t *args, size_t args_len, const char *key);

/*
 * Parse an unsigned integer from a MessagePack map.
 * Returns 0 on success, -1 if not found.
 */
int parse_uint_arg(const uint8_t *args, size_t args_len, const char *key, uint64_t *out);

#endif /* COMMANDS_H */
