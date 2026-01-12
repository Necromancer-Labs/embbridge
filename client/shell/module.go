/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * EDB shell module - main module definition and command registration
 */

package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/Necromancerlabs/gocmd2/pkg/shellapi"
	"github.com/spf13/cobra"
)

// EDBModule provides device interaction commands
type EDBModule struct {
	shell shellapi.ShellAPI
	proto *protocol.Protocol
	cwd   string
}

// NewEDBModule creates a new EDB module
func NewEDBModule(proto *protocol.Protocol) *EDBModule {
	return &EDBModule{
		proto: proto,
		cwd:   "/",
	}
}

// Name returns the module name
func (m *EDBModule) Name() string {
	return "edb"
}

// Initialize is called when the module is registered
func (m *EDBModule) Initialize(s shellapi.ShellAPI) {
	m.shell = s

	// Fetch initial cwd from agent
	resp, err := m.proto.Pwd()
	if err == nil && resp.OK {
		if path, ok := resp.Data["path"].(string); ok {
			m.cwd = path
		}
	}

	// Set initial prompt
	m.updatePrompt()
}

func (m *EDBModule) updatePrompt() {
	m.shell.SetPrompt(fmt.Sprintf("edb[%s]#", m.cwd))
}

// GetCommands returns all commands provided by this module
func (m *EDBModule) GetCommands() []*cobra.Command {
	commands := []*cobra.Command{}

	// ==========================================================================
	// Navigation commands
	// ==========================================================================

	// ls command
	lsCmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory contents",
		Run: func(cmd *cobra.Command, args []string) {
			path := m.cwd
			if len(args) > 0 {
				path = args[0]
			}
			m.doLs(path)
		},
	}
	commands = append(commands, lsCmd)

	// cd command
	cdCmd := &cobra.Command{
		Use:   "cd <path>",
		Short: "Change directory (supports relative paths like .. and .)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			m.doCd(args[0])
		},
	}
	commands = append(commands, cdCmd)

	// pwd command
	pwdCmd := &cobra.Command{
		Use:   "pwd",
		Short: "Print working directory",
		Run: func(cmd *cobra.Command, args []string) {
			m.doPwd()
		},
	}
	commands = append(commands, pwdCmd)

	// cat command
	catCmd := &cobra.Command{
		Use:   "cat <file>",
		Short: "Print file contents (absolute path required)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "file") {
				return
			}
			m.doCat(args[0])
		},
	}
	commands = append(commands, catCmd)

	// realpath command
	realpathCmd := &cobra.Command{
		Use:   "realpath <path>",
		Short: "Resolve path to canonical absolute form",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			m.doRealpath(args[0])
		},
	}
	commands = append(commands, realpathCmd)

	// ==========================================================================
	// System commands
	// ==========================================================================

	// uname command
	unameCmd := &cobra.Command{
		Use:   "uname",
		Short: "Print system information",
		Run: func(cmd *cobra.Command, args []string) {
			m.doUname()
		},
	}
	commands = append(commands, unameCmd)

	// ps command
	psCmd := &cobra.Command{
		Use:   "ps",
		Short: "List processes (tree view)",
		Run: func(cmd *cobra.Command, args []string) {
			m.doPs()
		},
	}
	commands = append(commands, psCmd)

	// ss command (socket statistics)
	ssCmd := &cobra.Command{
		Use:   "ss",
		Short: "List network connections (TCP/UDP with process info)",
		Run: func(cmd *cobra.Command, args []string) {
			m.doSs()
		},
	}
	commands = append(commands, ssCmd)

	// exec command
	execCmd := &cobra.Command{
		Use:   "exec <command>",
		Short: "Execute a binary on the device (no shell)",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmdStr := strings.Join(args, " ")
			m.doExec(cmdStr)
		},
	}
	commands = append(commands, execCmd)

	// kill-agent command
	killAgentCmd := &cobra.Command{
		Use:   "kill-agent",
		Short: "Kill the agent's parent process (bind mode only)",
		Run: func(cmd *cobra.Command, args []string) {
			m.doKillAgent()
		},
	}
	commands = append(commands, killAgentCmd)

	// reboot command
	rebootCmd := &cobra.Command{
		Use:   "reboot",
		Short: "Reboot the device",
		Run: func(cmd *cobra.Command, args []string) {
			m.doReboot()
		},
	}
	commands = append(commands, rebootCmd)

	// whoami command
	whoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show current user (uid/gid)",
		Run: func(cmd *cobra.Command, args []string) {
			m.doWhoami()
		},
	}
	commands = append(commands, whoamiCmd)

	// dmesg command
	dmesgCmd := &cobra.Command{
		Use:   "dmesg",
		Short: "Show kernel log messages",
		Run: func(cmd *cobra.Command, args []string) {
			m.doDmesg()
		},
	}
	commands = append(commands, dmesgCmd)

	// strings command
	stringsCmd := &cobra.Command{
		Use:   "strings <file>",
		Short: "Extract printable strings from a file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "file") {
				return
			}
			m.doStrings(args[0])
		},
	}
	commands = append(commands, stringsCmd)

	// cpuinfo command
	cpuinfoCmd := &cobra.Command{
		Use:   "cpuinfo",
		Short: "Show CPU information",
		Run: func(cmd *cobra.Command, args []string) {
			m.doCpuinfo()
		},
	}
	commands = append(commands, cpuinfoCmd)

	// mtd command
	mtdCmd := &cobra.Command{
		Use:   "mtd",
		Short: "List MTD (flash) partitions",
		Run: func(cmd *cobra.Command, args []string) {
			m.doMtd()
		},
	}
	commands = append(commands, mtdCmd)

	// ==========================================================================
	// File transfer commands
	// ==========================================================================

	// pull command (download from device)
	pullCmd := &cobra.Command{
		Use:   "pull <remote-file> [local-path]",
		Short: "Download a file from the device to your local machine",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			remotePath := args[0]
			if !requireAbsolutePath(remotePath, "remote-path") {
				return
			}
			localPath := filepath.Base(remotePath)
			if len(args) > 1 {
				localPath = args[1]
			}
			m.doGet(remotePath, localPath)
		},
	}
	commands = append(commands, pullCmd)

	// push command (upload to device)
	pushCmd := &cobra.Command{
		Use:   "push <local-file> <remote-path>",
		Short: "Upload a file from your local machine to the device",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[1], "remote-path") {
				return
			}
			m.doPut(args[0], args[1])
		},
	}
	commands = append(commands, pushCmd)

	// ==========================================================================
	// File operation commands
	// ==========================================================================

	// rm command
	rmCmd := &cobra.Command{
		Use:   "rm <path>",
		Short: "Remove a file or empty directory (absolute path required)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "path") {
				return
			}
			m.doRm(args[0])
		},
	}
	commands = append(commands, rmCmd)

	// mv command
	mvCmd := &cobra.Command{
		Use:   "mv <src> <dst>",
		Short: "Move or rename a file/directory (absolute paths required)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "src") {
				return
			}
			if !requireAbsolutePath(args[1], "dst") {
				return
			}
			m.doMv(args[0], args[1])
		},
	}
	commands = append(commands, mvCmd)

	// cp command
	cpCmd := &cobra.Command{
		Use:   "cp <src> <dst>",
		Short: "Copy a file (absolute paths required)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "src") {
				return
			}
			if !requireAbsolutePath(args[1], "dst") {
				return
			}
			m.doCp(args[0], args[1])
		},
	}
	commands = append(commands, cpCmd)

	// mkdir command
	mkdirCmd := &cobra.Command{
		Use:   "mkdir <path>",
		Short: "Create a directory (absolute path required)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[0], "path") {
				return
			}
			m.doMkdir(args[0])
		},
	}
	commands = append(commands, mkdirCmd)

	// chmod command
	chmodCmd := &cobra.Command{
		Use:   "chmod <mode> <path>",
		Short: "Change file permissions (absolute path required, e.g., chmod 755 /tmp/file)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if !requireAbsolutePath(args[1], "path") {
				return
			}
			m.doChmod(args[0], args[1])
		},
	}
	commands = append(commands, chmodCmd)

	return commands
}
