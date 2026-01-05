/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Listen command - wait for agent reverse connections
 */

package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/Necromancer-Labs/embbridge/client/shell"
)

var listenPort int

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for agent connections",
	Long: `Start listening for incoming agent connections (reverse mode).

When an agent connects, you'll be dropped into an interactive shell.

Examples:
  edb listen              # Listen on default port (1337)
  edb listen -p 4444      # Listen on port 4444`,
	RunE: runListen,
}

func init() {
	listenCmd.Flags().IntVarP(&listenPort, "port", "p", DefaultPort, "Port to listen on")
}

func runListen(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf(":%d", listenPort)

	fmt.Printf("Listening on %s...\n", addr)
	fmt.Println("Waiting for agent connection (Ctrl+C to cancel)")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start listening
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	// Accept in a goroutine so we can handle signals
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	// Wait for connection or signal
	select {
	case <-sigChan:
		fmt.Println("\nCancelled")
		return nil
	case err := <-errChan:
		return fmt.Errorf("accept failed: %w", err)
	case conn := <-connChan:
		fmt.Printf("Agent connected from %s\n", conn.RemoteAddr())
		return handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	// Create protocol handler
	proto := protocol.New(conn)

	// Perform handshake - expect hello from agent
	hello, err := proto.RecvHello()
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	if !hello.IsAgent {
		return fmt.Errorf("expected agent, got client")
	}

	fmt.Printf("Agent version: %d\n", hello.Version)

	// Send hello_ack
	if err := proto.SendHelloAck(); err != nil {
		return fmt.Errorf("failed to send hello_ack: %w", err)
	}

	// Start interactive shell
	return shell.RunShell(proto)
}
