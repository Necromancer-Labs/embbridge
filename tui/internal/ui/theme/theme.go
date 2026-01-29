// Package theme provides the visual styling for the Graveyard TUI.
//
// This package defines a consistent color palette and lipgloss styles
// used throughout the application for:
//   - Title bars and headers
//   - Status indicators (connected, busy, error, disconnected)
//   - List item selection
//   - Help text and keyboard shortcuts
//   - Shell prompt and error messages
//   - File browser (directories, files, symlinks)
//
// The theme uses a purple/violet color scheme branded for Necromancer Labs.
package theme

import "github.com/charmbracelet/lipgloss"

// Color palette - Necromancer Labs purple/violet theme
var (
	Primary   = lipgloss.Color("#8A2BE2") // Purple - brand color, used for accents
	Secondary = lipgloss.Color("#374151") // Dark gray - backgrounds
	Success   = lipgloss.Color("#22c55e") // Green - connected/success states
	Warning   = lipgloss.Color("#f59e0b") // Amber - busy/warning states
	Error     = lipgloss.Color("#ef4444") // Red - error/disconnected states
	Muted     = lipgloss.Color("#6b7280") // Gray - inactive/secondary text
	White     = lipgloss.Color("#ffffff") // White - primary text
	Black     = lipgloss.Color("#000000") // Black - backgrounds
)

// Styles - reusable lipgloss styles for UI components
var (
	// TitleStyle is used for the main title bar ("GRAVEYARD")
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Primary).
			Padding(0, 1)

	// StatusConnected indicates a connected/live device (green)
	StatusConnected = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	// StatusBusy indicates a device executing a command (amber)
	StatusBusy = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	// StatusError indicates a device with an error (red)
	StatusError = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// StatusDisconnected indicates a disconnected device (gray)
	StatusDisconnected = lipgloss.NewStyle().
				Foreground(Muted)

	// SelectedItem highlights the currently selected item in lists
	SelectedItem = lipgloss.NewStyle().
			Foreground(White).
			Background(Primary).
			Padding(0, 1)

	// NormalItem is the default style for unselected list items
	NormalItem = lipgloss.NewStyle().
			Foreground(White).
			Padding(0, 1)

	// BorderStyle is used for bordered containers
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary)

	// StatusBar is used for the bottom status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(Muted).
			Background(Secondary).
			Padding(0, 1)

	// HelpKey styles keyboard shortcut keys in help text
	HelpKey = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	// HelpDesc styles the description text for keyboard shortcuts
	HelpDesc = lipgloss.NewStyle().
			Foreground(Muted)

	// PromptStyle is used for the shell prompt and selection cursor
	PromptStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// MutedStyle is used for secondary/inactive text
	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// DirStyle colors directory names in file listings (purple, bold)
	DirStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// FileStyle colors regular file names in file listings (white)
	FileStyle = lipgloss.NewStyle().
			Foreground(White)

	// LinkStyle colors symbolic link names in file listings (cyan)
	LinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06b6d4")) // Cyan
)
