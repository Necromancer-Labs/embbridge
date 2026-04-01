/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Protocol handshake - hello/hello_ack exchange
 */

package protocol

import "fmt"

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
