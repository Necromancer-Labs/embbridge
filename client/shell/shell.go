/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Interactive shell using gocmd2
 */

package shell

import (
	"fmt"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/Necromancerlabs/gocmd2/pkg/shell"
)

// RunShell starts the interactive shell
func RunShell(proto *protocol.Protocol) error {
	// Create the shell
	sh, err := shell.NewShell("edb", `
	────────────────────────────────────────────────────────────
	embbridge  ·  Embedded Debug Bridge
	Type 'help' to list available commands
	────────────────────────────────────────────────────────────
	`)
	if err != nil {
		return fmt.Errorf("failed to create shell: %w", err)
	}
	defer sh.Close()

	// Register the EDB module
	edbModule := NewEDBModule(proto)
	sh.RegisterModule(edbModule)

	// Set history file
	sh.SetHistoryFile("/tmp/edb_history")

	// Run the shell
	sh.Run()

	return nil
}
