/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Test command - send a single command without TUI (for debugging)
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Necromancer-Labs/embbridge/client/protocol"
)

var testCmd = &cobra.Command{
	Use:    "test <host:port> <command>",
	Short:  "Send a test command (no TUI, for debugging)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(2),
	RunE:   runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	target := args[0]
	command := args[1]

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
	fmt.Println("Sending hello...")
	if err := proto.SendHello(); err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}

	// Expect hello_ack from agent
	fmt.Println("Waiting for hello_ack...")
	ack, err := proto.RecvHelloAck()
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	fmt.Printf("Agent version: %d\n", ack.Version)
	fmt.Printf("Sending command: %s\n", command)

	// Send the command
	var resp *protocol.Response

	switch command {
	case "ls":
		path := "."
		if len(args) > 2 {
			path = args[2]
		}
		resp, err = proto.Ls(path)
	case "pwd":
		resp, err = proto.Pwd()
	case "cd":
		if len(args) < 3 {
			return fmt.Errorf("cd requires a path argument")
		}
		resp, err = proto.Cd(args[2])
	case "cat":
		if len(args) < 3 {
			return fmt.Errorf("cat requires a path argument")
		}
		resp, err = proto.Cat(args[2])
	case "uname":
		resp, err = proto.Uname()
	default:
		return fmt.Errorf("unknown test command: %s", command)
	}

	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	// Pretty print response
	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response:\n%s\n", respJSON)

	return nil
}
