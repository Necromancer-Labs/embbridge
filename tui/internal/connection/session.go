package connection

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
)

// Session wraps the embbridge protocol for a single device connection
type Session struct {
	proto  *protocol.Protocol
	device *Device
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
}

// NewSession creates a session from an accepted connection
func NewSession(conn net.Conn, device *Device) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		proto:  protocol.New(conn),
		device: device,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Device returns the device associated with this session
func (s *Session) Device() *Device {
	return s.device
}

// Context returns the session's context
func (s *Session) Context() context.Context {
	return s.ctx
}

// Close terminates the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	s.cancel()
	return s.proto.Close()
}

// IsClosed returns whether the session is closed
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// PerformHandshakeAsClient performs handshake as the initiating party (connect mode)
// Client sends hello first, expects hello_ack back
func (s *Session) PerformHandshakeAsClient() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.proto.SendHello(); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	if _, err := s.proto.RecvHelloAck(); err != nil {
		return fmt.Errorf("receive hello_ack: %w", err)
	}

	return nil
}

// PerformHandshakeAsServer performs handshake as the listening party (listen mode)
// Expects hello from agent, responds with hello_ack
func (s *Session) PerformHandshakeAsServer() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.proto.RecvHello(); err != nil {
		return fmt.Errorf("receive hello: %w", err)
	}

	if err := s.proto.SendHelloAck(); err != nil {
		return fmt.Errorf("send hello_ack: %w", err)
	}

	return nil
}

// Initialize fetches device info (uname, whoami, pwd) after handshake
func (s *Session) Initialize() error {
	// Get system info
	resp, err := s.Uname()
	if err != nil {
		return fmt.Errorf("uname: %w", err)
	}
	if resp.OK {
		hostname, _ := resp.Data["nodename"].(string)
		kernel, _ := resp.Data["release"].(string)
		arch, _ := resp.Data["machine"].(string)
		os, _ := resp.Data["sysname"].(string)
		s.device.UpdateInfo(hostname, kernel, arch, os)
	}

	// Get user info
	resp, err = s.Whoami()
	if err != nil {
		return fmt.Errorf("whoami: %w", err)
	}
	if resp.OK {
		user, _ := resp.Data["user"].(string)
		uid := toInt(resp.Data["uid"])
		gid := toInt(resp.Data["gid"])
		s.device.UpdateUser(user, uid, gid)
	}

	// Get current directory
	resp, err = s.Pwd()
	if err != nil {
		return fmt.Errorf("pwd: %w", err)
	}
	if resp.OK {
		if path, ok := resp.Data["path"].(string); ok {
			s.device.SetCurrentDir(path)
		}
	}

	s.device.SetState(DeviceConnected)
	return nil
}

// Execute runs a command and returns the response (thread-safe)
func (s *Session) Execute(cmd string, args map[string]interface{}) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	if _, err := s.proto.SendRequest(cmd, args); err != nil {
		return nil, err
	}
	return s.proto.RecvResponse()
}

// Ls lists a directory
func (s *Session) Ls(path string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Ls(path)
}

// Pwd gets the current working directory
func (s *Session) Pwd() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Pwd()
}

// Cd changes the current directory
func (s *Session) Cd(path string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	resp, err := s.proto.Cd(path)
	if err != nil {
		return nil, err
	}
	if resp.OK {
		if newPath, ok := resp.Data["path"].(string); ok {
			s.device.SetCurrentDir(newPath)
		}
	}
	return resp, nil
}

// Cat reads a file
func (s *Session) Cat(path string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Cat(path)
}

// Uname gets system info
func (s *Session) Uname() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Uname()
}

// Whoami gets the current user
func (s *Session) Whoami() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Whoami()
}

// Ps gets process list
func (s *Session) Ps() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Ps()
}

// Ss gets network connections
func (s *Session) Ss() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Ss()
}

// Exec runs a command on the device
func (s *Session) Exec(command string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Exec(command)
}

// Dmesg gets kernel log
func (s *Session) Dmesg() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Dmesg()
}

// Rm removes a file or directory
func (s *Session) Rm(path string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Rm(path)
}

// Mv moves/renames a file
func (s *Session) Mv(src, dst string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Mv(src, dst)
}

// Cp copies a file
func (s *Session) Cp(src, dst string) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Cp(src, dst)
}

// Mkdir creates a directory
func (s *Session) Mkdir(path string, mode uint32) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Mkdir(path, mode)
}

// Chmod changes file permissions
func (s *Session) Chmod(path string, mode uint32) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Chmod(path, mode)
}

// Reboot reboots the device
func (s *Session) Reboot() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Reboot()
}

// Pull downloads a file with progress callback
func (s *Session) Pull(remotePath string, progress protocol.TransferProgress) ([]byte, int64, uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, 0, 0, fmt.Errorf("session closed")
	}

	return s.proto.Pull(remotePath, progress)
}

// Push uploads a file with progress callback
func (s *Session) Push(remotePath string, data []byte, mode uint32, progress protocol.TransferProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session closed")
	}

	return s.proto.Push(remotePath, data, mode, progress)
}

// IpAddr shows network interfaces
func (s *Session) IpAddr() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.IpAddr()
}

// IpRoute shows routing table
func (s *Session) IpRoute() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.IpRoute()
}

// Cpuinfo gets CPU information
func (s *Session) Cpuinfo() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Cpuinfo()
}

// Mtd lists MTD partitions
func (s *Session) Mtd() (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Mtd()
}

// Strings extracts printable strings from a file
func (s *Session) Strings(path string, minLen int) (*protocol.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("session closed")
	}

	return s.proto.Strings(path, minLen)
}

// Helper to convert interface{} to int
func toInt(v interface{}) int {
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
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
