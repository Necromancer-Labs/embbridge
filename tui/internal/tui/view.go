package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/Necromancer-Labs/embbridge-tui/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// colorOption represents a selectable highlight color
type colorOption struct {
	name string // Display name
	hex  string // Hex color code
}

// colorOptions defines the available highlight colors for devices.
// Purple (default) is first, followed by a rainbow of options.
var colorOptions = []colorOption{
	{"purple", "#8A2BE2"},  // Default - brand color
	{"red", "#ef4444"},     // Red
	{"orange", "#f97316"},  // Orange
	{"yellow", "#eab308"},  // Yellow
	{"green", "#22c55e"},   // Green
	{"cyan", "#06b6d4"},    // Cyan
	{"blue", "#3b82f6"},    // Blue
	{"pink", "#ec4899"},    // Pink
	{"white", "#ffffff"},   // White
}

// View renders the complete device list UI.
// The layout consists of:
//   - A bordered box containing the title bar and device list
//   - A help bar below the box (or connect dialog when active)
func (m Model) View() string {
	var content strings.Builder

	// Column header row
	header := fmt.Sprintf("  %-21s │ %-21s │ %-8s │ %-8s │ %s",
		"NAME", "ADDRESS", "ARCH", "STATUS", "UPTIME")
	content.WriteString(theme.MutedStyle.Render(header))
	content.WriteString("\n")

	// Header separator line
	content.WriteString(theme.MutedStyle.Render("────────────────────────┼───────────────────────┼──────────┼──────────┼────────"))
	content.WriteString("\n")

	// Device list or empty state message
	if len(m.devices) == 0 {
		content.WriteString(theme.MutedStyle.Render("No devices to haunt. Waiting for the dead to rise..."))
		content.WriteString("\n")
	} else {
		for i, device := range m.devices {
			content.WriteString(m.renderDevice(i, device))
			content.WriteString("\n")
		}
	}

	// Build the bordered box with rounded corners
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	// Title bar: app name, subtitle, and listen address
	title := theme.TitleStyle.Render(" GRAVEYARD ")
	subtitle := theme.MutedStyle.Render(" embbridge device manager")
	listenInfo := theme.StatusConnected.Render(fmt.Sprintf(" %s ", m.manager.ListenAddr()))
	titleBar := lipgloss.JoinHorizontal(lipgloss.Left, title, subtitle, listenInfo)

	// Combine title bar and device list inside the box
	fullContent := titleBar + "\n\n" + content.String()
	box := boxStyle.Render(fullContent)

	// Help bar or dialog below the box
	var helpBar string
	if m.showConnect {
		helpBar = m.renderConnectDialog()
	} else if m.showRename {
		helpBar = m.renderRenameDialog()
	} else if m.showArch {
		helpBar = m.renderArchDialog()
	} else if m.showColor {
		helpBar = m.renderColorPicker()
	} else {
		helpBar = m.renderHelp()
	}

	return box + "\n" + helpBar
}

// renderDevice renders a single device row with all columns.
// The row includes: cursor indicator, name, address, architecture, status, and uptime/last seen.
// The device name is always shown in the device's custom color (or default purple).
func (m Model) renderDevice(index int, device *connection.Device) string {
	// Get device's custom color (or default purple)
	deviceColor := device.GetCustomColor()
	if deviceColor == "" {
		deviceColor = string(theme.Primary) // Default purple
	}
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(deviceColor)).Bold(true)

	// Selection cursor indicator (► or empty space)
	cursor := "  "
	if index == m.cursor {
		cursor = nameStyle.Render("► ")
	}

	// Device name (truncated if too long) - always colored
	name := device.DisplayName()
	if len(name) > 20 {
		name = name[:17] + "..."
	}
	name = nameStyle.Render(fmt.Sprintf("%-20s", name))

	// Remote address (truncated if too long)
	addr := device.RemoteAddr
	if len(addr) > 21 {
		addr = addr[:18] + "..."
	}
	addr = fmt.Sprintf("%-21s", addr)

	// CPU architecture (uses DisplayArch which prefers custom over detected)
	arch := fmt.Sprintf("%-8s", device.DisplayArch())

	// Status badge (LIVE, BUSY, DEAD, etc.)
	// Note: renderStatus returns pre-padded string to handle ANSI codes
	status := renderStatus(device.GetState())

	// Time column: uptime if connected, "last seen" if disconnected
	var timeStr string
	if device.GetState() == connection.DeviceDisconnected {
		// Show time since last connection (how long ago it was seen)
		timeStr = fmt.Sprintf("%-6s", formatDuration(device.ConnectionDuration()))
	} else {
		// Show connection duration
		timeStr = fmt.Sprintf("%-6s", formatDuration(device.ConnectionDuration()))
	}

	// Build the complete row with column separators
	// Name is already styled, other columns use default color
	row := fmt.Sprintf("%s  │ %s │ %s │ %s │ %s", name, addr, arch, status, timeStr)

	return cursor + row
}

// renderStatus renders a colored status badge for the device state.
// Returns a pre-padded string because ANSI escape codes don't count toward
// fmt.Sprintf width calculations.
//
// Status badges:
//   - ◐ RISE  : Device is connecting (handshake in progress)
//   - ● LIVE  : Device is connected and ready
//   - ● BUSY  : Device is executing a command
//   - ↔ XFER  : Device is transferring a file
//   - ● ERR   : Device encountered an error
//   - ○ DEAD  : Device is disconnected
func renderStatus(state connection.DeviceState) string {
	switch state {
	case connection.DeviceConnecting:
		return theme.StatusBusy.Render("◐ RISE  ")
	case connection.DeviceConnected:
		return theme.StatusConnected.Render("● LIVE  ")
	case connection.DeviceBusy:
		return theme.StatusBusy.Render("● BUSY  ")
	case connection.DeviceTransferring:
		return theme.StatusBusy.Render("↔ XFER  ")
	case connection.DeviceError:
		return theme.StatusError.Render("● ERR   ")
	case connection.DeviceDisconnected:
		return theme.StatusDisconnected.Render("○ DEAD  ")
	default:
		return theme.MutedStyle.Render("? ???   ")
	}
}

// formatDuration formats a time.Duration as a compact human-readable string.
// Examples: "45s", "5m", "2h"
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// renderHelp renders the keyboard shortcut help bar.
// Shows available actions: navigation, connect, add, remove, rename, arch, color, quit.
func (m Model) renderHelp() string {
	help := []struct {
		key  string
		desc string
	}{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"enter", "connect"},
		{"c", "add"},
		{"d", "remove"},
		{"n", "name"},
		{"a", "arch"},
		{"h", "highlight"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range help {
		key := theme.HelpKey.Render(h.key)
		desc := theme.HelpDesc.Render(h.desc)
		parts = append(parts, fmt.Sprintf("%s %s", key, desc))
	}

	return strings.Join(parts, "  ")
}

// renderConnectDialog renders the address input dialog.
// Shown when user presses 'c' to add a new connection.
func (m Model) renderConnectDialog() string {
	label := theme.PromptStyle.Render("Connect to: ")
	input := m.connectInput.View()
	hint := theme.MutedStyle.Render(" (enter to connect, esc to cancel)")
	return label + input + hint
}

// renderRenameDialog renders the device rename input dialog.
// Shown when user presses 'n' to rename a device.
// The custom name overrides the device's hostname in the display.
// Enter an empty name to clear the custom name and revert to hostname.
func (m Model) renderRenameDialog() string {
	label := theme.PromptStyle.Render("Rename device: ")
	input := m.renameInput.View()
	hint := theme.MutedStyle.Render(" (enter to save, esc to cancel, empty to clear)")
	return label + input + hint
}

// renderArchDialog renders the architecture input dialog.
// Shown when user presses 'a' to set device architecture.
// The custom arch overrides the detected architecture in the display.
// Enter an empty value to clear and revert to detected arch.
func (m Model) renderArchDialog() string {
	label := theme.PromptStyle.Render("Set architecture: ")
	input := m.archInput.View()
	hint := theme.MutedStyle.Render(" (enter to save, esc to cancel, empty to clear)")
	return label + input + hint
}

// renderColorPicker renders the color selection picker.
// Shown when user presses 'h' to change device highlight color.
// Displays colored swatches with the current selection highlighted.
func (m Model) renderColorPicker() string {
	label := theme.PromptStyle.Render("Highlight color: ")

	var swatches []string
	for i, opt := range colorOptions {
		// Create a colored block for this option
		style := lipgloss.NewStyle().
			Background(lipgloss.Color(opt.hex)).
			Foreground(lipgloss.Color("#000000"))

		// Add selection indicator
		if i == m.colorCursor {
			// Selected: show name in the swatch
			swatches = append(swatches, style.Render(fmt.Sprintf(" %s ", opt.name)))
		} else {
			// Not selected: just show a colored block
			swatches = append(swatches, style.Render("   "))
		}
	}

	picker := strings.Join(swatches, " ")
	hint := theme.MutedStyle.Render(" (←/→ select, enter to save, esc to cancel)")

	return label + picker + hint
}
