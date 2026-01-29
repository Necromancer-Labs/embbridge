// Package shell provides the gocmd2-based interactive shell for device control.
//
// This package implements the shell portion of Graveyard's hybrid architecture.
// When a user selects a device from the TUI, this package takes over and provides
// a readline-based command interface with:
//   - Command history (persisted to ~/.graveyard_history)
//   - Tab completion for command names
//   - All device interaction commands (filesystem, system, network, etc.)
//
// The shell runs until the user types 'exit' or presses Ctrl+D, at which point
// control returns to the TUI device list.
package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/Necromancer-Labs/embbridge-tui/internal/shell/commands"
	"github.com/Necromancer-Labs/embbridge-tui/internal/ui/theme"
	"github.com/Necromancerlabs/gocmd2/pkg/shell"
	"github.com/spf13/cobra"
)

// RunShell runs the gocmd2 interactive shell for a connected device.
// This function blocks until the user exits the shell (via 'exit' command or Ctrl+D).
//
// The shell is initialized with:
//   - A banner showing device connection info
//   - Command history from ~/.graveyard_history
//   - Device and session stored in shell state (accessible to commands)
//   - All graveyard commands registered via the commands module
//   - Path tab completion for filesystem commands
//   - Custom exit command that returns to device list instead of terminating
func RunShell(device *connection.Device) {
	// Create banner with device connection info
	banner := formatBanner(device)

	// Create gocmd2 shell instance
	sh, err := shell.NewShell("graveyard", banner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create shell: %v\n", err)
		return
	}
	defer sh.Close()

	// Set history file for command recall (up/down arrows)
	homeDir, _ := os.UserHomeDir()
	historyFile := filepath.Join(homeDir, ".graveyard_history")
	sh.SetHistoryFile(historyFile)

	// Store device and session in shell's shared state
	// Commands retrieve these via Module.GetDevice() and Module.GetSession()
	sh.SetState("device", device)
	sh.SetState("session", device.GetSession())

	// Register the graveyard command module (all device commands)
	module := commands.NewModule()
	sh.RegisterModule(module)

	// Set up path completion for filesystem commands
	// This wraps the default command completer and adds remote path completion
	setupPathCompletion(sh)

	// Override gocmd2's default exit command
	// Default calls os.Exit(0), we want to return to device list instead
	overrideExitCommand(sh)

	// Set initial prompt showing device name and current directory
	updatePrompt(sh, device)

	// Run the REPL - blocks until user exits
	sh.Run()
}

// overrideExitCommand replaces gocmd2's core exit command with a custom one.
//
// The default exit command from gocmd2's core module calls os.Exit(0), which
// would terminate the entire program. Instead, we want 'exit' to just close
// the readline instance, which causes sh.Run() to return, allowing the main
// loop to show the device list again.
func overrideExitCommand(sh *shell.Shell) {
	rootCmd := sh.GetRootCmd()
	rl := sh.GetReadline()

	// Find and remove the existing exit command from core module
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "exit" {
			rootCmd.RemoveCommand(cmd)
			break
		}
	}

	// Add our custom exit command that closes readline gracefully
	exitCmd := &cobra.Command{
		Use:     "exit",
		Aliases: []string{"quit"},
		Short:   "Return to the crypt",
		Run: func(cmd *cobra.Command, args []string) {
			rl.Close() // Causes sh.Run() to return
		},
	}
	rootCmd.AddCommand(exitCmd)
}

// formatBanner creates the welcome banner shown when entering shell mode.
// Displays the device name, address, architecture, and user (if available).
func formatBanner(device *connection.Device) string {
	name := device.DisplayName()
	addr := device.RemoteAddr
	arch := device.Architecture
	user := device.User

	// Build styled banner with device info
	title := theme.TitleStyle.Render(" GRAVEYARD ")
	info := fmt.Sprintf("Haunting %s (%s)", name, addr)
	if arch != "" {
		info += fmt.Sprintf(" [%s]", arch)
	}
	if user != "" {
		info += fmt.Sprintf(" as %s", user)
	}

	return fmt.Sprintf("\n%s %s\n\nType 'help' for commands, 'exit' to return to the crypt.\n",
		title, info)
}

// setupPathCompletion configures tab completion for remote file paths.
// It wraps the default command completer with a PathCompleter that handles
// path arguments for filesystem commands like ls, cd, cat, etc.
func setupPathCompletion(sh *shell.Shell) {
	rl := sh.GetReadline()

	// Get the current completer (handles command names)
	cmdCompleter := rl.Config.AutoComplete

	// Create path completer that wraps the command completer
	pathCompleter := NewPathCompleter(sh, cmdCompleter)

	// Set the path completer as the new autocomplete handler
	rl.Config.AutoComplete = pathCompleter
}

// updatePrompt updates the shell prompt based on current device state.
// The prompt shows: graveyard[device-name]:/current/path#
func updatePrompt(sh *shell.Shell, device *connection.Device) {
	name := device.DisplayName()
	dir := device.GetCurrentDir()

	prompt := theme.PromptStyle.Render(fmt.Sprintf("graveyard[%s]:%s#", name, dir))
	sh.SetPrompt(prompt)
}
