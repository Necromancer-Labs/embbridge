/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Shell command - connect to agent and start interactive shell
 */

package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/Necromancer-Labs/embbridge/client/shell"
)

var shellCmd = &cobra.Command{
	Use:   "shell <host:port>",
	Short: "Connect to agent and start interactive shell",
	Long: `Connect to an agent running in bind mode and start an interactive shell.

Examples:
  edb shell 192.168.1.50:1337
  edb shell 192.168.1.50        # Uses default port 1337`,
	Args: cobra.ExactArgs(1),
	RunE: runShell,
}

func runShell(cmd *cobra.Command, args []string) error {
	target := args[0]

	// Add default port if not specified
	if !strings.Contains(target, ":") {
		target = fmt.Sprintf("%s:%d", target, DefaultPort)
	}

	fmt.Printf("Connecting to %s...\n", target)

	conn, err := net.Dial("tcp", target)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	fmt.Println("Connected")

	// Create protocol handler
	proto := protocol.New(conn)

	// In bind mode, client sends hello first
	if err := proto.SendHello(); err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}

	// Expect hello_ack from agent
	ack, err := proto.RecvHelloAck()
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	fmt.Printf("Agent version: %d\n", ack.Version)

	// Start interactive shell
	return shell.RunShell(proto)
}
