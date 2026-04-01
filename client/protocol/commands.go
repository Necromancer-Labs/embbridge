/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Protocol commands - high-level command methods
 */

package protocol

// Ls lists a directory
func (p *Protocol) Ls(path string) (*Response, error) {
	args := map[string]any{"path": path}
	if _, err := p.SendRequest("ls", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Pwd gets the current working directory
func (p *Protocol) Pwd() (*Response, error) {
	if _, err := p.SendRequest("pwd", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cd changes the current directory
func (p *Protocol) Cd(path string) (*Response, error) {
	args := map[string]any{"path": path}
	if _, err := p.SendRequest("cd", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Realpath resolves a path to its canonical absolute form
func (p *Protocol) Realpath(path string) (*Response, error) {
	args := map[string]any{"path": path}
	if _, err := p.SendRequest("realpath", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cat reads a file
func (p *Protocol) Cat(path string) (*Response, error) {
	args := map[string]any{"path": path}
	if _, err := p.SendRequest("cat", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Uname gets system info
func (p *Protocol) Uname() (*Response, error) {
	if _, err := p.SendRequest("uname", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Ps gets process list
func (p *Protocol) Ps() (*Response, error) {
	if _, err := p.SendRequest("ps", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Ss gets network connections (socket statistics)
func (p *Protocol) Ss() (*Response, error) {
	if _, err := p.SendRequest("ss", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// KillAgent kills the parent agent process (bind mode only)
func (p *Protocol) KillAgent() (*Response, error) {
	if _, err := p.SendRequest("kill-agent", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Kill sends a signal to a process. Default signal is 9 (SIGKILL).
func (p *Protocol) Kill(pid int, signal int) (*Response, error) {
	args := map[string]any{"pid": uint64(pid)}
	if signal != 0 {
		args["signal"] = uint64(signal)
	}
	if _, err := p.SendRequest("kill", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Exec runs a command
func (p *Protocol) Exec(command string) (*Response, error) {
	args := map[string]any{"command": command}
	if _, err := p.SendRequest("exec", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Reboot reboots the device
func (p *Protocol) Reboot() (*Response, error) {
	if _, err := p.SendRequest("reboot", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Whoami gets the current user
func (p *Protocol) Whoami() (*Response, error) {
	if _, err := p.SendRequest("whoami", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Dmesg gets kernel log messages
func (p *Protocol) Dmesg() (*Response, error) {
	if _, err := p.SendRequest("dmesg", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Strings extracts printable strings from a file
func (p *Protocol) Strings(path string, minLen int) (*Response, error) {
	args := map[string]any{"path": path}
	if minLen > 0 {
		args["min_len"] = minLen
	}
	if _, err := p.SendRequest("strings", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cpuinfo gets CPU information
func (p *Protocol) Cpuinfo() (*Response, error) {
	if _, err := p.SendRequest("cpuinfo", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Mtd lists MTD partitions
func (p *Protocol) Mtd() (*Response, error) {
	if _, err := p.SendRequest("mtd", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// IpAddr shows network interfaces
func (p *Protocol) IpAddr() (*Response, error) {
	if _, err := p.SendRequest("ip_addr", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// IpRoute shows routing table
func (p *Protocol) IpRoute() (*Response, error) {
	if _, err := p.SendRequest("ip_route", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Rm removes a file or empty directory
func (p *Protocol) Rm(path string) (*Response, error) {
	args := map[string]any{"path": path}
	if _, err := p.SendRequest("rm", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Mv moves/renames a file or directory
func (p *Protocol) Mv(src, dst string) (*Response, error) {
	args := map[string]any{"src": src, "dst": dst}
	if _, err := p.SendRequest("mv", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cp copies a file
func (p *Protocol) Cp(src, dst string) (*Response, error) {
	args := map[string]any{"src": src, "dst": dst}
	if _, err := p.SendRequest("cp", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Mkdir creates a directory
func (p *Protocol) Mkdir(path string, mode uint32) (*Response, error) {
	args := map[string]any{"path": path, "mode": uint64(mode)}
	if _, err := p.SendRequest("mkdir", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Chmod changes file permissions
func (p *Protocol) Chmod(path string, mode uint32) (*Response, error) {
	args := map[string]any{"path": path, "mode": uint64(mode)}
	if _, err := p.SendRequest("chmod", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}
