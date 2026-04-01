/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Port forwarding: tunnel TCP/UDP connections through the agent
 *
 * Architecture:
 *   LocalApp <--TCP/UDP--> Client <--embbridge--> Agent <--TCP/UDP--> Target
 *
 * Two usage patterns:
 *
 *   Client shell (no concurrent readers):
 *     tunnel.Start(host, port)   // calls ForwardOpen directly
 *     tunnel.Stop()              // calls ForwardClose directly
 *
 *   TUI (heartbeat goroutine races with protocol calls):
 *     resp, _ := session.ForwardOpen(host, port, transport)  // mutex-protected
 *     tunnel.StartForwarding(resp.ID)
 *     ...
 *     tunnel.StopForwarding()
 *     session.ForwardClose()  // mutex-protected, dispatcher still active
 *     tunnel.Cleanup()
 */

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// =============================================================================
// Tunnel Message Types
// =============================================================================

// TunnelDataMsg carries data through the tunnel
type TunnelDataMsg struct {
	Type string `msgpack:"type"`
	ID   uint32 `msgpack:"id"`
	Data []byte `msgpack:"data"`
}

// =============================================================================
// Protocol Methods for Tunneling
// =============================================================================

// ForwardOpen opens a tunnel to the specified remote host:port through the agent.
// transport is "tcp" or "udp" (empty defaults to "tcp").
func (p *Protocol) ForwardOpen(host string, port uint16, transport string) (*Response, error) {
	args := map[string]any{
		"host": host,
		"port": uint64(port),
	}
	if transport == "udp" {
		args["proto"] = "udp"
	}
	if _, err := p.SendRequest("forward_open", args); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// ForwardClose closes the current tunnel
func (p *Protocol) ForwardClose() (*Response, error) {
	if _, err := p.SendRequest("forward_close", nil); err != nil {
		return nil, err
	}
	return p.RecvResponse()
}

// SendTunnelData sends data through the tunnel
func (p *Protocol) SendTunnelData(id uint32, data []byte) error {
	msg := TunnelDataMsg{
		Type: "tunnel_data",
		ID:   id,
		Data: data,
	}
	return p.Send(msg)
}

// SendTunnelReconnect tells the agent to close its target connection.
// On the next tunnel_data, the agent will auto-reconnect to the target.
// This prevents stale connections when a local client disconnects and
// a new one connects (e.g., SSH ^C then SSH again).
func (p *Protocol) SendTunnelReconnect(id uint32) error {
	msg := TunnelDataMsg{
		Type: "tunnel_reconnect",
		ID:   id,
	}
	p.debugf("sending tunnel_reconnect id=%d", id)
	return p.Send(msg)
}

// GetConn returns the underlying connection
func (p *Protocol) GetConn() net.Conn {
	return p.conn
}

// =============================================================================
// Forward Tunnel
// =============================================================================

// ForwardTunnel manages a port forward session (TCP or UDP)
type ForwardTunnel struct {
	proto     *Protocol
	tunnelID  uint32
	localAddr string
	transport string          // "tcp" or "udp"
	listener  net.Listener    // TCP: local listener
	localConn net.Conn        // TCP: current local connection
	udpConn   net.PacketConn  // UDP: local packet connection
	lastAddr  net.Addr        // UDP: last client address for responses
	running   int32
	mu        sync.Mutex
	wg        sync.WaitGroup
}

// NewForwardTunnel creates a new forward tunnel manager.
// transport is "tcp" or "udp".
func NewForwardTunnel(proto *Protocol, localAddr string, transport string) *ForwardTunnel {
	if transport == "" {
		transport = "tcp"
	}
	return &ForwardTunnel{
		proto:     proto,
		localAddr: localAddr,
		transport: transport,
	}
}

// Start opens a tunnel and begins forwarding.
// Calls ForwardOpen directly on the protocol — safe when there are no
// concurrent readers (e.g., client shell). For TUI with heartbeat goroutines,
// use session.ForwardOpen() + StartForwarding() instead.
func (ft *ForwardTunnel) Start(remoteHost string, remotePort uint16) error {
	resp, err := ft.proto.ForwardOpen(remoteHost, remotePort, ft.transport)
	if err != nil {
		return fmt.Errorf("forward_open request failed: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("forward_open failed: %s", resp.Error)
	}
	return ft.StartForwarding(resp.ID)
}

// StartForwarding begins forwarding with an already-established tunnel ID.
// Use this when ForwardOpen was done separately through a Session mutex.
func (ft *ForwardTunnel) StartForwarding(tunnelID uint32) error {
	ft.tunnelID = tunnelID

	// Start the message dispatcher BEFORE any goroutines that read.
	if err := ft.proto.StartDispatcher(); err != nil {
		return fmt.Errorf("start dispatcher: %w", err)
	}

	atomic.StoreInt32(&ft.running, 1)

	if ft.transport == "udp" {
		// UDP: listen for datagrams on local port
		udpConn, err := net.ListenPacket("udp", ft.localAddr)
		if err != nil {
			ft.proto.StopDispatcher()
			return fmt.Errorf("listen udp on %s: %w", ft.localAddr, err)
		}
		ft.udpConn = udpConn

		ft.wg.Add(1)
		go ft.udpLocalToRemote()
	} else {
		// TCP: accept loop for local connections
		listener, err := net.Listen("tcp", ft.localAddr)
		if err != nil {
			ft.proto.StopDispatcher()
			return fmt.Errorf("listen on %s: %w", ft.localAddr, err)
		}
		ft.listener = listener

		ft.wg.Add(1)
		go ft.acceptLoop()
	}

	ft.wg.Add(1)
	go ft.forwardRemoteToLocal()

	return nil
}

// Stop stops forwarding and sends forward_close to the agent.
// For use in single-threaded contexts (client shell) with no concurrent readers.
func (ft *ForwardTunnel) Stop() {
	ft.StopForwarding()
	ft.proto.ForwardClose()
	ft.Cleanup()
}

// StopForwarding stops accepting connections and forwarding data, but keeps
// the dispatcher alive so the caller can send forward_close through a mutex.
func (ft *ForwardTunnel) StopForwarding() {
	if !atomic.CompareAndSwapInt32(&ft.running, 1, 0) {
		return
	}

	if ft.listener != nil {
		ft.listener.Close()
	}
	if ft.udpConn != nil {
		ft.udpConn.Close()
	}

	ft.mu.Lock()
	if ft.localConn != nil {
		ft.localConn.Close()
		ft.localConn = nil
	}
	ft.mu.Unlock()
}

// Cleanup stops the dispatcher and waits for all goroutines to finish.
// Call after forward_close has been sent.
func (ft *ForwardTunnel) Cleanup() {
	ft.proto.StopDispatcher()
	ft.wg.Wait()
}

// =============================================================================
// TCP Local Forwarding
// =============================================================================

// acceptLoop handles incoming local TCP connections
func (ft *ForwardTunnel) acceptLoop() {
	defer ft.wg.Done()

	for atomic.LoadInt32(&ft.running) == 1 {
		ft.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

		conn, err := ft.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if atomic.LoadInt32(&ft.running) == 0 {
				return
			}
			continue
		}

		// Single-stream: close any existing connection
		ft.mu.Lock()
		hadPrevious := ft.localConn != nil
		if hadPrevious {
			ft.localConn.Close()
		}
		ft.localConn = conn
		ft.mu.Unlock()

		// Tell agent to close stale target connection so it reconnects fresh
		if hadPrevious {
			ft.proto.SendTunnelReconnect(ft.tunnelID)
		}

		ft.wg.Add(1)
		go ft.forwardLocalToRemote(conn)
	}
}

// forwardLocalToRemote reads from local TCP connection and sends to tunnel
func (ft *ForwardTunnel) forwardLocalToRemote(conn net.Conn) {
	defer ft.wg.Done()
	defer conn.Close()

	buf := make([]byte, 32768)
	for atomic.LoadInt32(&ft.running) == 1 {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if err := ft.proto.SendTunnelData(ft.tunnelID, buf[:n]); err != nil {
			return
		}
	}
}

// =============================================================================
// UDP Local Forwarding
// =============================================================================

// udpLocalToRemote reads datagrams from local UDP socket and sends to tunnel
func (ft *ForwardTunnel) udpLocalToRemote() {
	defer ft.wg.Done()

	buf := make([]byte, 65535) // Max UDP datagram
	for atomic.LoadInt32(&ft.running) == 1 {
		ft.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, addr, err := ft.udpConn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if atomic.LoadInt32(&ft.running) == 0 {
				return
			}
			continue
		}

		// Track source address for sending responses back
		ft.mu.Lock()
		ft.lastAddr = addr
		ft.mu.Unlock()

		if err := ft.proto.SendTunnelData(ft.tunnelID, buf[:n]); err != nil {
			return
		}
	}
}

// =============================================================================
// Remote to Local (shared by TCP and UDP)
// =============================================================================

// forwardRemoteToLocal reads tunnel_data from the dispatcher channel and writes to local
func (ft *ForwardTunnel) forwardRemoteToLocal() {
	defer ft.wg.Done()

	for atomic.LoadInt32(&ft.running) == 1 {
		tunnelMsg, err := ft.proto.RecvTunnelData()
		if err != nil {
			if atomic.LoadInt32(&ft.running) == 0 {
				return
			}
			atomic.StoreInt32(&ft.running, 0)
			return
		}

		ft.handleTunnelData(tunnelMsg.Data)
	}
}

// handleTunnelData writes data to the current local connection/socket
func (ft *ForwardTunnel) handleTunnelData(data []byte) {
	if ft.transport == "udp" {
		ft.mu.Lock()
		addr := ft.lastAddr
		conn := ft.udpConn
		ft.mu.Unlock()

		if conn == nil || addr == nil {
			return
		}
		if _, err := conn.WriteTo(data, addr); err != nil {
			ft.proto.debugf("UDP write error: %v", err)
		}
		return
	}

	// TCP path
	ft.mu.Lock()
	conn := ft.localConn
	ft.mu.Unlock()

	if conn == nil {
		return
	}
	if _, err := conn.Write(data); err != nil {
		ft.proto.debugf("TCP write error: %v", err)
	}
}

// =============================================================================
// Status
// =============================================================================

// IsRunning returns true if the tunnel is active
func (ft *ForwardTunnel) IsRunning() bool {
	return atomic.LoadInt32(&ft.running) == 1
}

// LocalAddr returns the local listening address
func (ft *ForwardTunnel) LocalAddr() string {
	if ft.udpConn != nil {
		return ft.udpConn.LocalAddr().String()
	}
	if ft.listener != nil {
		return ft.listener.Addr().String()
	}
	return ft.localAddr
}

// Transport returns "tcp" or "udp"
func (ft *ForwardTunnel) Transport() string {
	return ft.transport
}

// =============================================================================
// Raw Message Handling
// =============================================================================

// RecvRaw receives a raw MessagePack message without decoding.
// Used by the dispatcher to read from the socket.
func (p *Protocol) RecvRaw() ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lenBuf); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length > MaxMsgSize {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	if length == 0 {
		return nil, nil
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(p.conn, data); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return data, nil
}

// DecodeMessageType extracts the "type" field from a raw msgpack message
func DecodeMessageType(data []byte) (string, error) {
	var msg struct {
		Type string `msgpack:"type"`
	}
	if err := msgpack.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	return msg.Type, nil
}

// DecodeTunnelData decodes tunnel_data message from raw msgpack
func DecodeTunnelData(data []byte) (*TunnelDataMsg, error) {
	var msg TunnelDataMsg
	if err := msgpack.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeResponse decodes a response message from raw msgpack
func DecodeResponse(data []byte) (*Response, error) {
	var resp Response
	if err := msgpack.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
