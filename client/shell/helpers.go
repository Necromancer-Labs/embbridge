/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Helper functions for shell commands
 */

package shell

import (
	"fmt"
	"strings"
)

// requireAbsolutePath checks if a path is absolute and prints an error if not.
// Returns true if the path is valid (absolute), false otherwise.
func requireAbsolutePath(path, argName string) bool {
	if !strings.HasPrefix(path, "/") {
		fmt.Printf("Error: %s must be an absolute path (starting with /)\n", argName)
		fmt.Printf("  Got: %s\n", path)
		return false
	}
	return true
}

// toInt64 safely converts interface{} to int64, handling various numeric types
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// toString safely converts interface{} to string
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// formatSizeShort formats a byte size as a short string (e.g., "1K", "5M")
func formatSizeShort(b int64) string {
	if b == 0 {
		return "0"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f%c", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatMode formats a Unix file mode as a permission string (e.g., "rwxr-xr-x")
func formatMode(mode uint64) string {
	var s [9]byte
	for i := 0; i < 9; i++ {
		s[i] = '-'
	}

	if mode&0400 != 0 {
		s[0] = 'r'
	}
	if mode&0200 != 0 {
		s[1] = 'w'
	}
	if mode&0100 != 0 {
		s[2] = 'x'
	}
	if mode&0040 != 0 {
		s[3] = 'r'
	}
	if mode&0020 != 0 {
		s[4] = 'w'
	}
	if mode&0010 != 0 {
		s[5] = 'x'
	}
	if mode&0004 != 0 {
		s[6] = 'r'
	}
	if mode&0002 != 0 {
		s[7] = 'w'
	}
	if mode&0001 != 0 {
		s[8] = 'x'
	}

	return string(s[:])
}

// formatBytes formats a byte size as a human-readable string (e.g., "1.5 KB")
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
