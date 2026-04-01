/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Run command - execute a single command non-interactively
 *
 * Usage:
 *   edb run <host:port> <command> [args...]
 *
 * Examples:
 *   edb run 192.168.3.30:1337 uname
 *   edb run 192.168.3.30:1337 ls /tmp
 *   edb run 192.168.3.30:1337 pull /etc/passwd ./passwd
 *   edb run 192.168.3.30:1337 push /tmp/agent /var/tmp/agent
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <host:port> <command> [args...]",
	Short: "Run a single command on an agent (non-interactive)",
	Long: `Connect to an agent, run a single command, print the result, and exit.

Examples:
  edb run 192.168.3.30:1337 uname
  edb run 192.168.3.30:1337 ls /tmp
  edb run 192.168.3.30:1337 cat /etc/passwd
  edb run 192.168.3.30:1337 pull /etc/passwd ./passwd
  edb run 192.168.3.30:1337 push ./agent /var/tmp/agent
  edb run 192.168.3.30:1337 exec "ls -la /tmp"
  edb run 192.168.3.30 whoami`,
	Args: cobra.MinimumNArgs(2),
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	target := args[0]
	command := args[1]
	cmdArgs := args[2:]

	// Add default port if not specified
	if !strings.Contains(target, ":") {
		target = fmt.Sprintf("%s:%d", target, DefaultPort)
	}

	// Connect
	conn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", target, err)
	}
	defer conn.Close()

	// Handshake
	proto := protocol.New(conn)
	if err := proto.SendHello(); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	if _, err := proto.RecvHelloAck(); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	// Dispatch command
	return dispatchCommand(proto, command, cmdArgs)
}

func dispatchCommand(proto *protocol.Protocol, command string, args []string) error {
	switch command {

	// --- Navigation ---

	case "ls":
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		return runLs(proto, path)

	case "pwd":
		return runSimpleString(proto, proto.Pwd, "path")

	case "cd":
		if len(args) < 1 {
			return fmt.Errorf("cd requires a path argument")
		}
		return runSimpleString(proto, func() (*protocol.Response, error) { return proto.Cd(args[0]) }, "path")

	case "cat":
		if len(args) < 1 {
			return fmt.Errorf("cat requires a file argument")
		}
		return runCat(proto, args[0])

	case "realpath":
		if len(args) < 1 {
			return fmt.Errorf("realpath requires a path argument")
		}
		return runSimpleString(proto, func() (*protocol.Response, error) { return proto.Realpath(args[0]) }, "path")

	// --- System info ---

	case "uname":
		return runUname(proto)

	case "whoami":
		return runWhoami(proto)

	case "ps":
		return runJSON(proto, proto.Ps)

	case "ss":
		return runJSON(proto, proto.Ss)

	case "dmesg":
		return runDmesg(proto)

	case "cpuinfo":
		return runJSON(proto, proto.Cpuinfo)

	case "mtd":
		return runJSON(proto, proto.Mtd)

	case "ip":
		return runJSON(proto, proto.IpAddr)

	case "ip-route":
		return runJSON(proto, proto.IpRoute)

	case "strings":
		if len(args) < 1 {
			return fmt.Errorf("strings requires a file argument")
		}
		return runStrings(proto, args[0])

	// --- System control ---

	case "exec":
		if len(args) < 1 {
			return fmt.Errorf("exec requires a command argument")
		}
		return runExec(proto, strings.Join(args, " "))

	case "reboot":
		return runOK(proto, func() (*protocol.Response, error) { return proto.Reboot() })

	case "kill-agent":
		return runOK(proto, proto.KillAgent)

	case "kill":
		if len(args) < 1 {
			return fmt.Errorf("kill requires <pid> [signal]")
		}
		pid, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid pid: %s", args[0])
		}
		sig := 0
		if len(args) > 1 {
			sig, err = strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid signal: %s", args[1])
			}
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Kill(pid, sig) })

	// --- File operations ---

	case "rm":
		if len(args) < 1 {
			return fmt.Errorf("rm requires a path argument")
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Rm(args[0]) })

	case "mv":
		if len(args) < 2 {
			return fmt.Errorf("mv requires <src> <dst> arguments")
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Mv(args[0], args[1]) })

	case "cp":
		if len(args) < 2 {
			return fmt.Errorf("cp requires <src> <dst> arguments")
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Cp(args[0], args[1]) })

	case "mkdir":
		if len(args) < 1 {
			return fmt.Errorf("mkdir requires a path argument")
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Mkdir(args[0], 0755) })

	case "chmod":
		if len(args) < 2 {
			return fmt.Errorf("chmod requires <mode> <path> arguments")
		}
		mode, err := strconv.ParseUint(args[0], 8, 32)
		if err != nil {
			return fmt.Errorf("invalid mode %q: %w", args[0], err)
		}
		return runOK(proto, func() (*protocol.Response, error) { return proto.Chmod(args[1], uint32(mode)) })

	// --- File transfer ---

	case "pull":
		if len(args) < 1 {
			return fmt.Errorf("pull requires <remote-path> [local-path]")
		}
		localPath := filepath.Base(args[0])
		if len(args) > 1 {
			localPath = args[1]
		}
		return runPull(proto, args[0], localPath)

	case "push":
		if len(args) < 2 {
			return fmt.Errorf("push requires <local-file> <remote-path>")
		}
		return runPush(proto, args[0], args[1])

	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// =============================================================================
// Command runners
// =============================================================================

// runLs formats directory listing
func runLs(proto *protocol.Protocol, path string) error {
	resp, err := proto.Ls(path)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	entries, ok := resp.Data["entries"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response")
	}

	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string)

		switch entryType {
		case "dir":
			fmt.Printf("%s/\n", name)
		case "link":
			fmt.Printf("%s@\n", name)
		default:
			fmt.Println(name)
		}
	}
	return nil
}

// runCat prints file content
func runCat(proto *protocol.Protocol, path string) error {
	resp, err := proto.Cat(path)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	if content, ok := resp.Data["content"].([]byte); ok {
		fmt.Print(string(content))
		if len(content) > 0 && content[len(content)-1] != '\n' {
			fmt.Println()
		}
	}
	return nil
}

// runUname prints system info
func runUname(proto *protocol.Protocol) error {
	resp, err := proto.Uname()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	sysname, _ := resp.Data["sysname"].(string)
	nodename, _ := resp.Data["nodename"].(string)
	release, _ := resp.Data["release"].(string)
	machine, _ := resp.Data["machine"].(string)

	fmt.Printf("%s %s %s %s\n", sysname, nodename, release, machine)
	return nil
}

// runWhoami prints user info
func runWhoami(proto *protocol.Protocol) error {
	resp, err := proto.Whoami()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	user, _ := resp.Data["user"].(string)
	uid := toRunInt(resp.Data["uid"])
	gid := toRunInt(resp.Data["gid"])

	fmt.Printf("%s (uid=%d gid=%d)\n", user, uid, gid)
	return nil
}

// runDmesg prints kernel log
func runDmesg(proto *protocol.Protocol) error {
	resp, err := proto.Dmesg()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	if content, ok := resp.Data["log"].(string); ok {
		fmt.Print(content)
	}
	return nil
}

// runStrings prints extracted strings
func runStrings(proto *protocol.Protocol, path string) error {
	resp, err := proto.Strings(path, 4)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	if strs, ok := resp.Data["strings"].([]interface{}); ok {
		for _, s := range strs {
			if str, ok := s.(string); ok {
				fmt.Println(str)
			}
		}
	}
	return nil
}

// runExec prints command output
func runExec(proto *protocol.Protocol, command string) error {
	resp, err := proto.Exec(command)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	if stdout, ok := resp.Data["stdout"].(string); ok && stdout != "" {
		fmt.Print(stdout)
	}
	if stderr, ok := resp.Data["stderr"].(string); ok && stderr != "" {
		fmt.Fprintf(os.Stderr, "%s", stderr)
	}
	return nil
}

// runPull downloads a file from the device
func runPull(proto *protocol.Protocol, remotePath, localPath string) error {
	start := time.Now()

	data, _, mode, err := proto.Pull(remotePath, func(transferred, total int64) {
		pct := float64(transferred) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "\r  %d / %d bytes (%.0f%%)", transferred, total, pct)
	})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("create %s: %w", localPath, err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", localPath, err)
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "\r  %d bytes -> %s (%v)\n", len(data), localPath, elapsed.Round(time.Millisecond))
	return nil
}

// runPush uploads a file to the device
func runPush(proto *protocol.Protocol, localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", localPath, err)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}
	mode := uint32(info.Mode().Perm())

	start := time.Now()
	err = proto.Push(remotePath, data, mode, func(transferred, total int64) {
		pct := float64(transferred) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "\r  %d / %d bytes (%.0f%%)", transferred, total, pct)
	})
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "\r  %s -> %s (%d bytes, %v)\n", localPath, remotePath, len(data), elapsed.Round(time.Millisecond))
	return nil
}

// =============================================================================
// Generic runners
// =============================================================================

// runSimpleString runs a command and prints a single string field
func runSimpleString(proto *protocol.Protocol, fn func() (*protocol.Response, error), field string) error {
	resp, err := fn()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	if val, ok := resp.Data[field].(string); ok {
		fmt.Println(val)
	}
	return nil
}

// runJSON runs a command and dumps response data as JSON
func runJSON(proto *protocol.Protocol, fn func() (*protocol.Response, error)) error {
	resp, err := fn()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	out, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// runOK runs a command and just reports success/failure
func runOK(proto *protocol.Protocol, fn func() (*protocol.Response, error)) error {
	resp, err := fn()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// toRunInt converts interface{} to int (for this package)
func toRunInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
