/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Protocol core - wire format, message types, and connection handling
 */

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	Version      = 1
	MaxMsgSize   = 16 * 1024 * 1024 // 16 MB
	DefaultChunk = 64 * 1024        // 64 KB
)

// Protocol handles the wire protocol for embbridge
type Protocol struct {
	conn   net.Conn
	mu     sync.Mutex // protects writes
	nextID uint32
	Debug  bool // enable debug logging (set EDB_DEBUG=1)

	// Dispatcher state (only active during tunnel forwarding)
	dispatchMu     sync.Mutex
	dispatching    bool
	responseCh     chan []byte   // receives "resp" and "data" messages
	tunnelCh       chan []byte   // receives "tunnel_data" messages
	dispatcherDone chan struct{} // closed when dispatcher goroutine exits
}

// New creates a new Protocol handler
func New(conn net.Conn) *Protocol {
	return &Protocol{
		conn:   conn,
		nextID: 1,
		Debug:  os.Getenv("EDB_DEBUG") != "",
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

// debugf prints a debug message if debugging is enabled
func (p *Protocol) debugf(format string, args ...any) {
	if p.Debug {
		fmt.Printf("[edb] "+format+"\n", args...)
	}
}

// =============================================================================
// Wire Format
// =============================================================================

// Send sends a MessagePack-encoded message with length prefix
func (p *Protocol) Send(v any) error {
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

	if _, err := p.conn.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// Recv receives a MessagePack-encoded message with length prefix.
// When the dispatcher is active, reads from the response channel instead of the socket.
func (p *Protocol) Recv(v any) error {
	if p.isDispatching() {
		p.debugf("recv: from dispatcher channel")
		data, ok := <-p.responseCh
		if !ok {
			return fmt.Errorf("dispatcher stopped")
		}
		return msgpack.Unmarshal(data, v)
	}

	p.debugf("recv: direct socket read")
	return p.recvDirect(v)
}

// recvDirect reads a message directly from the socket (used when dispatcher is inactive)
func (p *Protocol) recvDirect(v any) error {
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
// Core Message Types
// =============================================================================

// Request is a command request from client to agent
type Request struct {
	Type string         `msgpack:"type"`
	ID   uint32         `msgpack:"id"`
	Cmd  string         `msgpack:"cmd"`
	Args map[string]any `msgpack:"args"`
}

// Response is a command response from agent to client
type Response struct {
	Type  string         `msgpack:"type"`
	ID    uint32         `msgpack:"id"`
	OK    bool           `msgpack:"ok"`
	Data  map[string]any `msgpack:"data,omitempty"`
	Error string         `msgpack:"error,omitempty"`
}

// SendRequest sends a command request
func (p *Protocol) SendRequest(cmd string, args map[string]any) (uint32, error) {
	id := p.NextID()
	req := Request{
		Type: "req",
		ID:   id,
		Cmd:  cmd,
		Args: args,
	}
	p.debugf("send: cmd=%s id=%d", cmd, id)
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

	p.debugf("recv: resp id=%d ok=%v", resp.ID, resp.OK)
	return &resp, nil
}

// =============================================================================
// Helpers
// =============================================================================

// toInt64 safely converts any to int64
func toInt64(v any) int64 {
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
