package commands

// Output formatting functions for command results.
// These functions transform raw agent response data into human-readable output
// with proper styling using lipgloss.

import (
	"fmt"
	"strings"

	"github.com/Necromancer-Labs/embbridge-tui/internal/ui/theme"
)

// FormatLsOutput formats directory listing output from the agent.
// Expects data to contain "entries" array with objects having:
//   - name: filename
//   - type: "dir", "file", "link", or "other"
//   - mode: numeric permissions (e.g., 0755)
//   - size: file size in bytes
//
// Returns formatted output with permissions, size, and colored filename.
func FormatLsOutput(data map[string]interface{}) string {
	entries, ok := data["entries"].([]interface{})
	if !ok {
		return ""
	}

	var b strings.Builder
	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string) // "dir", "file", "link", "other"
		modeNum := toInt64(entry["mode"])       // numeric mode (0-777)
		size := toInt64(entry["size"])

		// Format mode from numeric to rwxrwxrwx string
		modeStr := formatMode(entryType, uint32(modeNum))

		// Format size as human-readable (e.g., "1.5K", "2.3M")
		sizeStr := formatSize(size)

		// Format name with color based on type
		var styledName string
		switch entryType {
		case "dir":
			styledName = theme.DirStyle.Render(name + "/")
		case "link":
			styledName = theme.LinkStyle.Render(name)
		default:
			styledName = theme.FileStyle.Render(name)
		}

		b.WriteString(fmt.Sprintf("%s  %8s  %s\n", modeStr, sizeStr, styledName))
	}
	return b.String()
}

// formatMode converts numeric permission mode to Unix-style string (e.g., "-rwxr-xr-x").
// The first character indicates the entry type: 'd' for directory, 'l' for link, '-' for file.
func formatMode(entryType string, mode uint32) string {
	// Determine type character
	var typeChar byte
	switch entryType {
	case "dir":
		typeChar = 'd'
	case "link":
		typeChar = 'l'
	default:
		typeChar = '-'
	}

	// Convert 3-bit permission value to rwx string
	permChars := func(perm uint32) string {
		r := "-"
		w := "-"
		x := "-"
		if perm&4 != 0 {
			r = "r"
		}
		if perm&2 != 0 {
			w = "w"
		}
		if perm&1 != 0 {
			x = "x"
		}
		return r + w + x
	}

	// Extract owner, group, other permissions from mode
	owner := permChars((mode >> 6) & 7)
	group := permChars((mode >> 3) & 7)
	other := permChars(mode & 7)

	return string(typeChar) + owner + group + other
}

// FormatPsOutput formats process list output from the agent.
// Expects data to contain "processes" array with objects having:
//   - pid: process ID
//   - ppid: parent process ID
//   - state: process state (R, S, Z, etc.)
//   - cmdline: command line string
//
// Returns formatted table with header.
func FormatPsOutput(data map[string]interface{}) string {
	processes, ok := data["processes"].([]interface{})
	if !ok {
		return ""
	}

	var b strings.Builder

	// Table header
	header := fmt.Sprintf("%-8s %-8s %-5s %s\n", "PID", "PPID", "STATE", "CMD")
	b.WriteString(theme.MutedStyle.Render(header))

	// Process rows
	for _, p := range processes {
		proc, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		pid := toInt64(proc["pid"])
		ppid := toInt64(proc["ppid"])
		state, _ := proc["state"].(string)
		cmdline, _ := proc["cmdline"].(string)

		b.WriteString(fmt.Sprintf("%-8d %-8d %-5s %s\n", pid, ppid, state, cmdline))
	}
	return b.String()
}

// FormatSsOutput formats socket/network connection output from the agent.
// Expects data to contain "connections" array with objects having:
//   - proto: protocol (tcp, udp)
//   - state: connection state (ESTABLISHED, LISTEN, etc.)
//   - local_addr, local_port: local endpoint
//   - remote_addr, remote_port: remote endpoint
//   - pid, process: associated process info
//
// Returns formatted table with header.
func FormatSsOutput(data map[string]interface{}) string {
	connections, ok := data["connections"].([]interface{})
	if !ok {
		return ""
	}

	var b strings.Builder

	// Table header
	header := fmt.Sprintf("%-8s %-12s %-24s %-24s %s\n", "PROTO", "STATE", "LOCAL", "REMOTE", "PID/PROCESS")
	b.WriteString(theme.MutedStyle.Render(header))

	// Connection rows
	for _, c := range connections {
		conn, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		proto, _ := conn["proto"].(string)
		state, _ := conn["state"].(string)
		localAddr, _ := conn["local_addr"].(string)
		localPort := toInt64(conn["local_port"])
		remoteAddr, _ := conn["remote_addr"].(string)
		remotePort := toInt64(conn["remote_port"])
		pid := toInt64(conn["pid"])
		process, _ := conn["process"].(string)

		// Format address:port pairs
		local := fmt.Sprintf("%s:%d", localAddr, localPort)
		remote := fmt.Sprintf("%s:%d", remoteAddr, remotePort)

		// Format PID/process column
		pidProc := "-"
		if pid > 0 {
			pidProc = fmt.Sprintf("%d/%s", pid, process)
		}

		b.WriteString(fmt.Sprintf("%-8s %-12s %-24s %-24s %s\n", proto, state, local, remote, pidProc))
	}
	return b.String()
}

// FormatUnameOutput formats system information output from the agent.
// Expects data to contain uname fields:
//   - sysname: OS name (e.g., "Linux")
//   - nodename: hostname
//   - release: kernel version
//   - version: kernel build info
//   - machine: architecture (e.g., "mips", "armv7l")
//
// Returns formatted key: value pairs.
func FormatUnameOutput(data map[string]interface{}) string {
	var b strings.Builder

	// Field definitions with display labels
	fields := []struct {
		key   string
		label string
	}{
		{"sysname", "System"},
		{"nodename", "Hostname"},
		{"release", "Kernel"},
		{"version", "Version"},
		{"machine", "Architecture"},
	}

	for _, f := range fields {
		if val, ok := data[f.key].(string); ok && val != "" {
			label := theme.MutedStyle.Render(f.label + ":")
			b.WriteString(fmt.Sprintf("%s %s\n", label, val))
		}
	}
	return b.String()
}

// FormatWhoamiOutput formats user information output from the agent.
// Expects data to contain:
//   - user: username
//   - uid: user ID
//   - gid: group ID
//
// Returns formatted user identity string.
func FormatWhoamiOutput(data map[string]interface{}) string {
	user, _ := data["user"].(string)
	uid := toInt64(data["uid"])
	gid := toInt64(data["gid"])

	return fmt.Sprintf("%s (uid=%d, gid=%d)\n", user, uid, gid)
}

// FormatCpuinfoOutput formats CPU information output from the agent.
// The agent sends raw /proc/cpuinfo content as binary data in "content" field.
// Returns the content as-is (no additional formatting).
func FormatCpuinfoOutput(data map[string]interface{}) string {
	// Agent sends "content" as binary (raw /proc/cpuinfo)
	if content, ok := data["content"].([]byte); ok {
		return string(content)
	}
	// Also try string in case msgpack decodes it differently
	if content, ok := data["content"].(string); ok {
		return content
	}
	return ""
}

// FormatMtdOutput formats MTD partition information output from the agent.
// The agent sends raw /proc/mtd content as binary data in "content" field.
// Returns the content as-is (no additional formatting).
func FormatMtdOutput(data map[string]interface{}) string {
	// Agent sends "content" as binary (raw /proc/mtd)
	if content, ok := data["content"].([]byte); ok {
		return string(content)
	}
	if content, ok := data["content"].(string); ok {
		return content
	}
	return ""
}

// FormatIpAddrOutput formats network interface information output from the agent.
// The agent sends pre-formatted text as binary data in "content" field.
// Returns the content as-is (no additional formatting).
func FormatIpAddrOutput(data map[string]interface{}) string {
	// Agent sends "content" as binary (pre-formatted text)
	if content, ok := data["content"].([]byte); ok {
		return string(content)
	}
	if content, ok := data["content"].(string); ok {
		return content
	}
	return ""
}

// FormatIpRouteOutput formats routing table output from the agent.
// The agent sends pre-formatted text as binary data in "content" field.
// Returns the content as-is (no additional formatting).
func FormatIpRouteOutput(data map[string]interface{}) string {
	// Agent sends "content" as binary (pre-formatted text)
	if content, ok := data["content"].([]byte); ok {
		return string(content)
	}
	if content, ok := data["content"].(string); ok {
		return content
	}
	return ""
}

// formatSize formats a byte size as a human-readable string.
// Examples: "512" (bytes), "1.5K", "2.3M", "1.0G"
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.1fG", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1fM", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1fK", float64(size)/KB)
	default:
		return fmt.Sprintf("%d", size)
	}
}

// toInt64 safely converts an interface{} value to int64.
// Handles all numeric types that msgpack might decode to.
// Returns 0 if the value cannot be converted.
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
