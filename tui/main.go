// Package main provides the entry point for Graveyard, the embbridge TUI.
//
// Graveyard uses a hybrid architecture:
//   - Bubble Tea: Visual TUI for device list selection (internal/tui)
//   - gocmd2: Readline-based shell for device interaction (internal/shell)
//
// The main loop alternates between these two modes:
//  1. Device list (Bubble Tea) - user selects a device
//  2. Shell (gocmd2) - user interacts with the selected device
//  3. On shell exit, return to device list
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/Necromancer-Labs/embbridge-tui/internal/shell"
	"github.com/Necromancer-Labs/embbridge-tui/internal/tui"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 1337, "Port to listen on for agent connections")
	flag.Parse()

	// Create connection manager - handles all device connections and state
	manager := connection.NewManager()

	// Start TCP listener for incoming agent connections
	listenAddr := fmt.Sprintf(":%d", *port)
	if err := manager.Listen(listenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen on %s: %v\n", listenAddr, err)
		os.Exit(1)
	}
	defer manager.Stop()

	fmt.Printf("Graveyard listening on %s\n", listenAddr)

	// Main loop: alternate between device list and shell
	// This is the core of the hybrid architecture
	for {
		// Run Bubble Tea device list, blocks until user selects a device or quits
		device := tui.RunDeviceList(manager)
		if device == nil {
			// User quit (pressed 'q' or ctrl+c)
			break
		}

		// Run gocmd2 shell for selected device, blocks until user exits shell
		shell.RunShell(device)
		// When shell exits (user types 'exit'), loop back to device list
	}

	fmt.Println("Rest in peace...")
}
