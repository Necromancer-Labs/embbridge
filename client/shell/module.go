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
	"strconv"
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

	// kill command
	killCmd := &cobra.Command{
		Use:   "kill <pid> [signal]",
		Short: "Send a signal to a process (default: 9/SIGKILL)",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Error: invalid pid: %s\n", args[0])
				return
			}
			sig := 0 // let agent default to SIGKILL
			if len(args) > 1 {
				sig, err = strconv.Atoi(args[1])
				if err != nil {
					fmt.Printf("Error: invalid signal: %s\n", args[1])
					return
				}
			}
			m.doKill(pid, sig)
		},
	}
	commands = append(commands, killCmd)

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

	// ip command
	ipCmd := &cobra.Command{
		Use:   "ip",
		Short: "Show network interfaces",
		Run: func(cmd *cobra.Command, args []string) {
			m.doIpAddr()
		},
	}
	commands = append(commands, ipCmd)

	// ip-route command
	ipRouteCmd := &cobra.Command{
		Use:   "ip-route",
		Short: "Show routing table",
		Run: func(cmd *cobra.Command, args []string) {
			m.doIpRoute()
		},
	}
	commands = append(commands, ipRouteCmd)

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

	// ==========================================================================
	// Port forwarding commands
	// ==========================================================================

	// forward command
	forwardCmd := &cobra.Command{
		Use:   "forward-tcp <localport> <remotehost> <remoteport>",
		Short: "Open a TCP port forward tunnel through the agent",
		Long: `Forward local TCP port to a remote host via the agent.

Examples:
  forward-tcp 8080 192.168.1.100 80     Access neighbor device's web UI at localhost:8080
  forward-tcp 8554 localhost 554        Access device's RTSP stream at localhost:8554
  forward-tcp 9000 10.0.0.1 22          SSH to another device via localhost:9000`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			localPort, err := parsePort(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			remotePort, err := parsePort(args[2])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			m.doForward(localPort, args[1], remotePort)
		},
	}
	commands = append(commands, forwardCmd)

	// forward-stop command
	forwardStopCmd := &cobra.Command{
		Use:   "forward-stop",
		Short: "Close the active port forward tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			m.doForwardStop()
		},
	}
	commands = append(commands, forwardStopCmd)

	// forward-status command
	forwardStatusCmd := &cobra.Command{
		Use:   "forward-status",
		Short: "Show the status of the active port forward tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			m.doForwardStatus()
		},
	}
	commands = append(commands, forwardStatusCmd)

	// forward-udp command
	forwardUDPCmd := &cobra.Command{
		Use:   "forward-udp <localport> <remotehost> <remoteport>",
		Short: "Open a UDP port forward tunnel through the agent",
		Long: `Forward local UDP port to a remote host via the agent.

Examples:
  forward-udp 1900 239.255.255.250 1900   Inspect UPnP/SSDP on the LAN
  forward-udp 5353 224.0.0.251 5353        Inspect mDNS on the LAN`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			localPort, err := parsePort(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			remotePort, err := parsePort(args[2])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			m.doForwardUDP(localPort, args[1], remotePort)
		},
	}
	commands = append(commands, forwardUDPCmd)

	return commands
}
