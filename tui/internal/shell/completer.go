// Package shell provides the gocmd2-based interactive shell for device control.
package shell

import (
	"path/filepath"
	"strings"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/Necromancerlabs/gocmd2/pkg/shellapi"
	"github.com/chzyer/readline"
)

// PathCompleter provides tab completion for file paths on the remote device.
// It wraps the default command completer and adds path completion for commands
// that take file/directory arguments.
type PathCompleter struct {
	shell shellapi.ShellAPI

	// cmdCompleter handles command name completion (fallback)
	cmdCompleter readline.AutoCompleter

	// pathCmds maps command names to argument positions that are paths
	// e.g., "mv" -> [0, 1] means both first and second args are paths
	pathCmds map[string][]int
}

// NewPathCompleter creates a completer that handles both commands and paths.
// The cmdCompleter is used for command name completion (first word).
func NewPathCompleter(shell shellapi.ShellAPI, cmdCompleter readline.AutoCompleter) *PathCompleter {
	return &PathCompleter{
		shell:        shell,
		cmdCompleter: cmdCompleter,
		pathCmds: map[string][]int{
			// Filesystem commands
			"ls":    {0},    // ls [path]
			"cd":    {0},    // cd <path>
			"cat":   {0},    // cat <file>
			"rm":    {0},    // rm <path>
			"mv":    {0, 1}, // mv <src> <dst>
			"cp":    {0, 1}, // cp <src> <dst>
			"mkdir": {0},    // mkdir <path>
			"chmod": {1},    // chmod <mode> <path>

			// Transfer commands (remote paths only)
			"pull": {0}, // pull <remote> [local]
			"push": {1}, // push <local> [remote]

			// Misc
			"strings": {0}, // strings <file>
		},
	}
}

// Do implements readline.AutoCompleter interface.
// Called when user presses TAB.
func (c *PathCompleter) Do(line []rune, pos int) ([][]rune, int) {
	// Get the text up to cursor position
	lineStr := string(line[:pos])

	// Split into words (simple split, doesn't handle quotes perfectly)
	parts := strings.Fields(lineStr)

	// If no input or still typing first word, use command completer
	if len(parts) == 0 {
		return c.cmdCompleter.Do(line, pos)
	}

	// Check if we're still on the first word (no space after it)
	if len(parts) == 1 && !strings.HasSuffix(lineStr, " ") {
		return c.cmdCompleter.Do(line, pos)
	}

	// We're completing an argument
	cmd := parts[0]

	// Check if this command takes path arguments
	pathPositions, ok := c.pathCmds[cmd]
	if !ok {
		// Not a path command, no completion
		return nil, 0
	}

	// Determine which argument we're completing
	// If line ends with space, we're starting a new argument
	argIndex := len(parts) - 1
	if strings.HasSuffix(lineStr, " ") {
		argIndex = len(parts)
	}

	// argIndex is 1-based (parts[0] is command, parts[1] is first arg)
	// pathPositions is 0-based (0 = first arg)
	argPos := argIndex - 1

	// Check if this argument position should be path-completed
	isPathArg := false
	for _, p := range pathPositions {
		if p == argPos {
			isPathArg = true
			break
		}
	}

	if !isPathArg {
		return nil, 0
	}

	// Get the partial path being typed
	partial := ""
	if argIndex < len(parts) {
		partial = parts[argIndex]
	}

	// Complete the path
	return c.completePath(partial)
}

// completePath queries the remote device and returns matching paths.
func (c *PathCompleter) completePath(partial string) ([][]rune, int) {
	// Get session from shell state
	session := c.getSession()
	if session == nil {
		return nil, 0
	}

	// Split partial into directory and prefix
	// "/etc/pas" -> dir="/etc", prefix="pas"
	// "/etc/" -> dir="/etc/", prefix=""
	// "foo" -> dir=".", prefix="foo"
	// "" -> dir=".", prefix=""
	var dir, prefix string

	if partial == "" {
		dir = "."
		prefix = ""
	} else if strings.HasSuffix(partial, "/") {
		dir = partial
		prefix = ""
	} else {
		dir = filepath.Dir(partial)
		prefix = filepath.Base(partial)
		// Handle case where partial is just a name with no dir
		if dir == "." && !strings.HasPrefix(partial, "./") && !strings.HasPrefix(partial, "/") {
			// Keep dir as "." for relative paths like "foo"
		}
	}

	// Query the remote device for directory listing
	resp, err := session.Ls(dir)
	if err != nil || !resp.OK {
		return nil, 0
	}

	// Get entries from response
	entries, ok := resp.Data["entries"].([]interface{})
	if !ok {
		return nil, 0
	}

	// Filter entries that match the prefix
	var matches [][]rune

	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string)

		// Skip . and .. entries
		if name == "." || name == ".." {
			continue
		}

		// Check if name matches prefix
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Build the completion suffix
		// We return the part that needs to be added after what's typed
		suffix := name[len(prefix):]

		// Add trailing slash for directories
		if entryType == "dir" {
			suffix += "/"
		}

		matches = append(matches, []rune(suffix))
	}

	// Return matches and length of prefix (how many chars to replace)
	return matches, len(prefix)
}

// getSession retrieves the device session from shell state.
func (c *PathCompleter) getSession() *connection.Session {
	if val, ok := c.shell.GetState("session"); ok {
		if session, ok := val.(*connection.Session); ok {
			return session
		}
	}
	return nil
}
