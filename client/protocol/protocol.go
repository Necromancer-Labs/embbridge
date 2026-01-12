/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Protocol handling - MessagePack encoding/decoding and wire format
 */

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	Version       = 1
	MaxMsgSize    = 16 * 1024 * 1024 // 16 MB
	DefaultChunk  = 64 * 1024        // 64 KB
)

// Protocol handles the wire protocol for embbridge
type Protocol struct {
	conn   net.Conn
	mu     sync.Mutex
	nextID uint32
}

// New creates a new Protocol handler
func New(conn net.Conn) *Protocol {
	return &Protocol{
		conn:   conn,
		nextID: 1,
	}
}

// Close closes the underlying connection
func (p *Protocol) Close() error {
	return p.conn.Close()
}

// NextID returns the next request ID
func (p *Protocol) NextID() uint32 {
	return atomic.AddUint32(&p.nextID, 1)
}

// =============================================================================
// Wire Format
// =============================================================================

// Send sends a MessagePack-encoded message with length prefix
func (p *Protocol) Send(v interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := msgpack.Marshal(v)
	if err != nil {
		return fmt.Errorf("msgpack encode: %w", err)
	}

	if len(data) > MaxMsgSize {
		return fmt.Errorf("message too large: %d bytes", len(data))
	}

	// Send length prefix (4 bytes, big-endian)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := p.conn.Write(lenBuf); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	// Send payload
	if _, err := p.conn.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// Recv receives a MessagePack-encoded message with length prefix
func (p *Protocol) Recv(v interface{}) error {
	// Read length prefix
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lenBuf); err != nil {
		return fmt.Errorf("read length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length > MaxMsgSize {
		return fmt.Errorf("message too large: %d bytes", length)
	}

	if length == 0 {
		return nil
	}

	// Read payload
	data := make([]byte, length)
	if _, err := io.ReadFull(p.conn, data); err != nil {
		return fmt.Errorf("read payload: %w", err)
	}

	if err := msgpack.Unmarshal(data, v); err != nil {
		return fmt.Errorf("msgpack decode: %w", err)
	}

	return nil
}

// =============================================================================
// Message Types
// =============================================================================

// HelloMsg is the initial handshake message
type HelloMsg struct {
	Type    string `msgpack:"type"`
	Version int    `msgpack:"version"`
	IsAgent bool   `msgpack:"agent"`
}

// HelloAckMsg is the handshake response
type HelloAckMsg struct {
	Type    string `msgpack:"type"`
	Version int    `msgpack:"version"`
	IsAgent bool   `msgpack:"agent"`
}

// Request is a command request from client to agent
type Request struct {
	Type string                 `msgpack:"type"`
	ID   uint32                 `msgpack:"id"`
	Cmd  string                 `msgpack:"cmd"`
	Args map[string]interface{} `msgpack:"args"`
}

// Response is a command response from agent to client
type Response struct {
	Type  string                 `msgpack:"type"`
	ID    uint32                 `msgpack:"id"`
	OK    bool                   `msgpack:"ok"`
	Data  map[string]interface{} `msgpack:"data,omitempty"`
	Error string                 `msgpack:"error,omitempty"`
}

// =============================================================================
// Handshake
// =============================================================================

// SendHello sends a hello message (client initiating)
func (p *Protocol) SendHello() error {
	msg := HelloMsg{
		Type:    "hello",
		Version: Version,
		IsAgent: false,
	}
	return p.Send(msg)
}

// SendHelloAck sends a hello_ack message
func (p *Protocol) SendHelloAck() error {
	msg := HelloAckMsg{
		Type:    "hello_ack",
		Version: Version,
		IsAgent: false,
	}
	return p.Send(msg)
}

// RecvHello receives and validates a hello message
func (p *Protocol) RecvHello() (*HelloMsg, error) {
	var msg HelloMsg
	if err := p.Recv(&msg); err != nil {
		return nil, err
	}

	if msg.Type != "hello" {
		return nil, fmt.Errorf("expected hello, got %s", msg.Type)
	}

	return &msg, nil
}

// RecvHelloAck receives and validates a hello_ack message
func (p *Protocol) RecvHelloAck() (*HelloAckMsg, error) {
	var msg HelloAckMsg
	if err := p.Recv(&msg); err != nil {
		return nil, err
	}

	if msg.Type != "hello_ack" {
		return nil, fmt.Errorf("expected hello_ack, got %s", msg.Type)
	}

	return &msg, nil
}

// =============================================================================
// Commands
// =============================================================================

// SendRequest sends a command request
func (p *Protocol) SendRequest(cmd string, args map[string]interface{}) (uint32, error) {
	id := p.NextID()
	req := Request{
		Type: "req",
		ID:   id,
		Cmd:  cmd,
		Args: args,
	}
	return id, p.Send(req)
}

// RecvResponse receives a command response
func (p *Protocol) RecvResponse() (*Response, error) {
	var resp Response
	if err := p.Recv(&resp); err != nil {
		return nil, err
	}

	if resp.Type != "resp" {
		return nil, fmt.Errorf("expected resp, got %s", resp.Type)
	}

	return &resp, nil
}

// =============================================================================
// High-Level Commands
// =============================================================================

// Ls lists a directory
func (p *Protocol) Ls(path string) (*Response, error) {
	args := map[string]interface{}{"path": path}
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
	args := map[string]interface{}{"path": path}
	if _, err := p.SendRequest("cd", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Realpath resolves a path to its canonical absolute form
func (p *Protocol) Realpath(path string) (*Response, error) {
	args := map[string]interface{}{"path": path}
	if _, err := p.SendRequest("realpath", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cat reads a file
func (p *Protocol) Cat(path string) (*Response, error) {
	args := map[string]interface{}{"path": path}
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

// Exec runs a command
func (p *Protocol) Exec(command string) (*Response, error) {
	args := map[string]interface{}{"command": command}
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
	args := map[string]interface{}{"path": path}
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

// Rm removes a file or empty directory
func (p *Protocol) Rm(path string) (*Response, error) {
	args := map[string]interface{}{"path": path}
	if _, err := p.SendRequest("rm", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Mv moves/renames a file or directory
func (p *Protocol) Mv(src, dst string) (*Response, error) {
	args := map[string]interface{}{"src": src, "dst": dst}
	if _, err := p.SendRequest("mv", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Cp copies a file
func (p *Protocol) Cp(src, dst string) (*Response, error) {
	args := map[string]interface{}{"src": src, "dst": dst}
	if _, err := p.SendRequest("cp", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Mkdir creates a directory
func (p *Protocol) Mkdir(path string, mode uint32) (*Response, error) {
	args := map[string]interface{}{"path": path, "mode": uint64(mode)}
	if _, err := p.SendRequest("mkdir", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// Chmod changes file permissions
func (p *Protocol) Chmod(path string, mode uint32) (*Response, error) {
	args := map[string]interface{}{"path": path, "mode": uint64(mode)}
	if _, err := p.SendRequest("chmod", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// =============================================================================
// Chunked Transfer Types
// =============================================================================

// DataMsg is a chunk of data for file transfer
type DataMsg struct {
	Type string `msgpack:"type"`
	ID   uint32 `msgpack:"id"`
	Seq  uint32 `msgpack:"seq"`
	Data []byte `msgpack:"data"`
	Done bool   `msgpack:"done"`
}

// TransferProgress is called during file transfers with progress info
type TransferProgress func(transferred, total int64)

// =============================================================================
// Pull (Download) with Progress
// =============================================================================

// Pull downloads a file from the device with progress reporting
func (p *Protocol) Pull(remotePath string, progress TransferProgress) ([]byte, int64, uint32, error) {
	args := map[string]interface{}{"path": remotePath}
	if _, err := p.SendRequest("pull", args); err != nil {
		return nil, 0, 0, err
	}

	// Receive initial response with file info
	resp, err := p.RecvResponse()
	if err != nil {
		return nil, 0, 0, err
	}

	if !resp.OK {
		return nil, 0, 0, fmt.Errorf("%s", resp.Error)
	}

	// Extract file size and mode
	size := toInt64(resp.Data["size"])
	mode := toInt64(resp.Data["mode"])

	// Receive data chunks
	var data []byte
	var transferred int64

	for {
		var chunk DataMsg
		if err := p.Recv(&chunk); err != nil {
			return nil, 0, 0, fmt.Errorf("receive chunk: %w", err)
		}

		if chunk.Type != "data" {
			return nil, 0, 0, fmt.Errorf("expected data, got %s", chunk.Type)
		}

		data = append(data, chunk.Data...)
		transferred += int64(len(chunk.Data))

		if progress != nil {
			progress(transferred, size)
		}

		if chunk.Done {
			break
		}
	}

	return data, size, uint32(mode), nil
}

// =============================================================================
// Push (Upload) with Progress
// =============================================================================

// SendData sends a data chunk
func (p *Protocol) SendData(id, seq uint32, data []byte, done bool) error {
	msg := DataMsg{
		Type: "data",
		ID:   id,
		Seq:  seq,
		Data: data,
		Done: done,
	}
	return p.Send(msg)
}

// Push uploads a file to the device with progress reporting
func (p *Protocol) Push(remotePath string, data []byte, mode uint32, progress TransferProgress) error {
	args := map[string]interface{}{
		"path": remotePath,
		"size": uint64(len(data)),
		"mode": uint64(mode),
	}
	id, err := p.SendRequest("push", args)
	if err != nil {
		return err
	}

	// Receive OK response
	resp, err := p.RecvResponse()
	if err != nil {
		return err
	}

	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	// Send data in chunks
	total := int64(len(data))
	var seq uint32
	var transferred int64

	for transferred < total {
		end := transferred + DefaultChunk
		if end > total {
			end = total
		}

		chunk := data[transferred:end]
		done := (end >= total)

		if err := p.SendData(id, seq, chunk, done); err != nil {
			return fmt.Errorf("send chunk: %w", err)
		}

		transferred = end
		seq++

		if progress != nil {
			progress(transferred, total)
		}
	}

	return nil
}

// =============================================================================
// Helpers
// =============================================================================

// toInt64 safely converts interface{} to int64, handling various integer types
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
