/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Message dispatcher - routes incoming messages by type during port forwarding
 *
 * When a tunnel is active, a single reader goroutine reads all messages
 * from the socket and routes them to channels by type:
 *   - "tunnel_data" -> tunnelCh  (consumed by ForwardTunnel)
 *   - "resp"/"data" -> responseCh (consumed by commands via Recv)
 *
 * This prevents concurrent socket reads between the tunnel and commands.
 */

package protocol

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// isDispatching returns true if the dispatcher is running
func (p *Protocol) isDispatching() bool {
	p.dispatchMu.Lock()
	defer p.dispatchMu.Unlock()
	return p.dispatching
}

// StartDispatcher begins the background message reader goroutine.
func (p *Protocol) StartDispatcher() error {
	p.dispatchMu.Lock()
	defer p.dispatchMu.Unlock()

	if p.dispatching {
		return fmt.Errorf("dispatcher already active")
	}

	p.responseCh = make(chan []byte, 16)
	p.tunnelCh = make(chan []byte, 64)
	p.dispatcherDone = make(chan struct{})
	p.dispatching = true

	// Clear any residual deadline
	p.conn.SetReadDeadline(time.Time{})

	p.debugf("dispatcher: started")
	go p.dispatchLoop()
	return nil
}

// StopDispatcher shuts down the background reader goroutine.
func (p *Protocol) StopDispatcher() {
	p.dispatchMu.Lock()
	if !p.dispatching {
		p.dispatchMu.Unlock()
		return
	}
	p.dispatching = false
	p.dispatchMu.Unlock()

	p.debugf("dispatcher: stopping")

	// Wait for dispatcher to notice the flag and exit (checks every 500ms)
	<-p.dispatcherDone

	// Clear deadline for subsequent direct reads
	p.conn.SetReadDeadline(time.Time{})

	// Drain and close channels
	close(p.responseCh)
	close(p.tunnelCh)
	for range p.responseCh {
	}
	for range p.tunnelCh {
	}

	p.debugf("dispatcher: stopped")
}

// dispatchLoop is the single reader goroutine that routes messages by type
func (p *Protocol) dispatchLoop() {
	defer close(p.dispatcherDone)

	for {
		p.dispatchMu.Lock()
		active := p.dispatching
		p.dispatchMu.Unlock()
		if !active {
			return
		}

		// Short deadline so we can check the stop flag periodically
		p.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		data, err := p.RecvRaw()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue // timeout, check stop flag
			}
			// Check if we were told to stop while reading
			p.dispatchMu.Lock()
			active = p.dispatching
			p.dispatchMu.Unlock()
			if !active {
				return
			}
			p.debugf("dispatcher: read error: %v", err)
			return
		}

		if data == nil {
			continue
		}

		msgType, err := DecodeMessageType(data)
		if err != nil {
			p.debugf("dispatcher: decode type error: %v", err)
			continue
		}

		p.debugf("dispatcher: routing %s (%d bytes)", msgType, len(data))

		switch msgType {
		case "tunnel_data":
			select {
			case p.tunnelCh <- data:
			default:
				// tunnel channel full, drop to avoid blocking
			}
		case "resp", "data":
			p.responseCh <- data
		default:
			p.debugf("dispatcher: unknown type %q, dropping", msgType)
		}
	}
}

// RecvTunnelData reads a tunnel_data message from the dispatcher channel.
// Blocks until data is available or the channel is closed.
func (p *Protocol) RecvTunnelData() (*TunnelDataMsg, error) {
	data, ok := <-p.tunnelCh
	if !ok {
		return nil, fmt.Errorf("tunnel closed")
	}
	return DecodeTunnelData(data)
}
