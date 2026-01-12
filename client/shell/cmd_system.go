/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * System commands: uname, ps, ss, exec
 */

package shell

import (
	"fmt"
	"sort"
	"strings"
)

func (m *EDBModule) doUname() {
	resp, err := m.proto.Uname()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	sysname, _ := resp.Data["sysname"].(string)
	nodename, _ := resp.Data["nodename"].(string)
	release, _ := resp.Data["release"].(string)
	machine, _ := resp.Data["machine"].(string)

	fmt.Printf("%s %s %s %s\n", sysname, nodename, release, machine)
}

// processInfo represents a single process for tree rendering
type processInfo struct {
	pid     int
	ppid    int
	name    string
	state   string
	cmdline string
}

func (m *EDBModule) doPs() {
	resp, err := m.proto.Ps()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	procs, ok := resp.Data["processes"].([]interface{})
	if !ok {
		fmt.Println("Error: invalid response")
		return
	}

	// Parse processes into our structure
	processes := make(map[int]*processInfo)
	children := make(map[int][]int)

	for _, p := range procs {
		proc, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		info := &processInfo{
			pid:     int(toInt64(proc["pid"])),
			ppid:    int(toInt64(proc["ppid"])),
			name:    toString(proc["name"]),
			state:   toString(proc["state"]),
			cmdline: toString(proc["cmdline"]),
		}

		processes[info.pid] = info
		children[info.ppid] = append(children[info.ppid], info.pid)
	}

	// Sort children by PID for consistent output
	for ppid := range children {
		sort.Ints(children[ppid])
	}

	// Find root processes (ppid=0 or ppid not in our list)
	var roots []int
	for pid, info := range processes {
		if info.ppid == 0 || processes[info.ppid] == nil {
			roots = append(roots, pid)
		}
	}
	sort.Ints(roots)

	// Print tree header
	fmt.Printf("%-7s %-7s %-5s %s\n", "PID", "PPID", "STATE", "COMMAND")
	fmt.Println(strings.Repeat("-", 80))

	// Render tree recursively - roots get special treatment (no branch)
	for _, pid := range roots {
		m.printProcessNode(processes, children, pid, "", true)
	}
}

// printProcessNode recursively prints the process tree with box-drawing characters
func (m *EDBModule) printProcessNode(processes map[int]*processInfo, children map[int][]int, pid int, prefix string, isRoot bool) {
	info := processes[pid]
	if info == nil {
		return
	}

	// Format cmdline (truncate if too long)
	cmdline := info.cmdline
	maxCmdLen := 80 - len(prefix) - 25 // account for PID/PPID/STATE columns
	if maxCmdLen < 20 {
		maxCmdLen = 20
	}
	if len(cmdline) > maxCmdLen {
		cmdline = cmdline[:maxCmdLen-3] + "..."
	}

	// Print this process
	fmt.Printf("%-7d %-7d %-5s %s%s\n",
		info.pid, info.ppid, info.state, prefix, cmdline)

	// Print children with tree branches
	kids := children[pid]
	for i, childPid := range kids {
		isLast := (i == len(kids)-1)

		// Determine branch character for child
		var branch string
		var childPrefix string
		if isLast {
			branch = "└── "
			childPrefix = prefix + "    "
		} else {
			branch = "├── "
			childPrefix = prefix + "│   "
		}

		// Print the child with branch
		m.printProcessNodeWithBranch(processes, children, childPid, prefix+branch, childPrefix)
	}
}

// printProcessNodeWithBranch prints a node that has a branch prefix, then recurses for children
func (m *EDBModule) printProcessNodeWithBranch(processes map[int]*processInfo, children map[int][]int, pid int, displayPrefix, childPrefix string) {
	info := processes[pid]
	if info == nil {
		return
	}

	// Format cmdline
	cmdline := info.cmdline
	maxCmdLen := 80 - len(displayPrefix) - 25
	if maxCmdLen < 20 {
		maxCmdLen = 20
	}
	if len(cmdline) > maxCmdLen {
		cmdline = cmdline[:maxCmdLen-3] + "..."
	}

	// Print this process with its branch
	fmt.Printf("%-7d %-7d %-5s %s%s\n",
		info.pid, info.ppid, info.state, displayPrefix, cmdline)

	// Print children
	kids := children[pid]
	for i, childPid := range kids {
		isLast := (i == len(kids)-1)

		var branch string
		var nextChildPrefix string
		if isLast {
			branch = "└── "
			nextChildPrefix = childPrefix + "    "
		} else {
			branch = "├── "
			nextChildPrefix = childPrefix + "│   "
		}

		m.printProcessNodeWithBranch(processes, children, childPid, childPrefix+branch, nextChildPrefix)
	}
}

func (m *EDBModule) doSs() {
	resp, err := m.proto.Ss()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	conns, ok := resp.Data["connections"].([]interface{})
	if !ok {
		fmt.Println("Error: invalid response")
		return
	}

	// Print header
	fmt.Printf("%-6s %-22s %-22s %-12s %7s  %s\n",
		"PROTO", "LOCAL ADDRESS", "REMOTE ADDRESS", "STATE", "PID", "PROCESS")
	fmt.Println(strings.Repeat("-", 90))

	for _, c := range conns {
		conn, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		proto := toString(conn["proto"])
		localAddr := toString(conn["local_addr"])
		localPort := int(toInt64(conn["local_port"]))
		remoteAddr := toString(conn["remote_addr"])
		remotePort := int(toInt64(conn["remote_port"]))
		state := toString(conn["state"])
		pid := int(toInt64(conn["pid"]))
		process := toString(conn["process"])

		// Format addresses with ports
		local := fmt.Sprintf("%s:%d", localAddr, localPort)
		remote := fmt.Sprintf("%s:%d", remoteAddr, remotePort)

		// Format PID/process
		var pidStr string
		if pid > 0 {
			pidStr = fmt.Sprintf("%d", pid)
		} else {
			pidStr = "-"
		}

		if process == "" {
			process = "-"
		}

		fmt.Printf("%-6s %-22s %-22s %-12s %7s  %s\n",
			proto, local, remote, state, pidStr, process)
	}
}

func (m *EDBModule) doExec(command string) {
	resp, err := m.proto.Exec(command)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if stdout, ok := resp.Data["stdout"].([]byte); ok && len(stdout) > 0 {
		fmt.Print(string(stdout))
	}
	if stderr, ok := resp.Data["stderr"].([]byte); ok && len(stderr) > 0 {
		fmt.Printf("\033[31m%s\033[0m", string(stderr)) // Red for stderr
	}
}

func (m *EDBModule) doKillAgent() {
	resp, err := m.proto.KillAgent()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	killedPid := int(toInt64(resp.Data["killed_pid"]))
	fmt.Printf("Agent parent process (pid %d) killed\n", killedPid)
}

func (m *EDBModule) doReboot() {
	resp, err := m.proto.Reboot()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Println("Device is rebooting...")
}

func (m *EDBModule) doWhoami() {
	resp, err := m.proto.Whoami()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	user, _ := resp.Data["user"].(string)
	uid := int(toInt64(resp.Data["uid"]))
	gid := int(toInt64(resp.Data["gid"]))

	fmt.Printf("%s (uid=%d, gid=%d)\n", user, uid, gid)
}

func (m *EDBModule) doDmesg() {
	resp, err := m.proto.Dmesg()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if log, ok := resp.Data["log"].([]byte); ok {
		fmt.Print(string(log))
	}
}

func (m *EDBModule) doStrings(path string) {
	resp, err := m.proto.Strings(path, 4) // default min_len = 4
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if content, ok := resp.Data["content"].([]byte); ok {
		fmt.Print(string(content))
	}
}

func (m *EDBModule) doCpuinfo() {
	resp, err := m.proto.Cpuinfo()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if content, ok := resp.Data["content"].([]byte); ok {
		fmt.Print(string(content))
	}
}

func (m *EDBModule) doMtd() {
	resp, err := m.proto.Mtd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if content, ok := resp.Data["content"].([]byte); ok {
		fmt.Print(string(content))
	}
	fmt.Println("\nTip: Use 'pull /dev/mtdX' to download a partition")
}
