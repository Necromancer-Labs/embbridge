// Package tui provides the Bubble Tea-based device list interface.
//
// This package implements the visual TUI portion of Graveyard's hybrid architecture.
// It displays a list of connected devices and allows the user to:
//   - Navigate the device list with arrow keys or vim bindings
//   - Select a device to enter shell mode
//   - Initiate new outbound connections
//   - Disconnect devices
//
// The TUI exits when a device is selected (returning control to main.go which
// then starts the shell) or when the user quits.
package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the device list view.
// It maintains the current state of the TUI including:
//   - The list of devices from the connection manager
//   - Current cursor position
//   - Selected device (set when user presses Enter)
//   - Connect dialog state for adding new connections
type Model struct {
	manager  *connection.Manager  // Reference to connection manager for device data
	devices  []*connection.Device // Cached list of devices (refreshed on tick/events)
	cursor   int                  // Current cursor position in device list
	selected *connection.Device   // Device selected by user (nil until Enter pressed)
	quit     bool                 // True if user quit (q/ctrl+c) vs selected a device

	// Connect dialog state
	showConnect  bool            // True when connect dialog is visible
	connectInput textinput.Model // Text input for entering address

	// Rename dialog state
	showRename  bool            // True when rename dialog is visible
	renameInput textinput.Model // Text input for entering new name

	// Arch dialog state
	showArch  bool            // True when arch dialog is visible
	archInput textinput.Model // Text input for entering architecture

	// Color picker state
	showColor   bool // True when color picker is visible
	colorCursor int  // Current selection in color picker

	// Terminal dimensions (updated on WindowSizeMsg)
	width  int
	height int
}

// tickMsg is sent periodically to refresh the device list.
// This ensures the UI stays up-to-date even without explicit events.
type tickMsg time.Time

// deviceEventMsg wraps a device event from the connection manager.
// Events include: connect, disconnect, state change, etc.
type deviceEventMsg connection.DeviceEvent

// NewModel creates a new device list model with the given connection manager.
// The model starts with the current device list and inactive dialogs.
func NewModel(manager *connection.Manager) Model {
	// Initialize text input for connect dialog
	connectTI := textinput.New()
	connectTI.Placeholder = "192.168.1.1:1337"
	connectTI.CharLimit = 64

	// Initialize text input for rename dialog
	renameTI := textinput.New()
	renameTI.Placeholder = "new-device-name"
	renameTI.CharLimit = 32

	// Initialize text input for arch dialog
	archTI := textinput.New()
	archTI.Placeholder = "mips/arm/x86"
	archTI.CharLimit = 16

	return Model{
		manager:      manager,
		devices:      manager.Devices(),
		connectInput: connectTI,
		renameInput:  renameTI,
		archInput:    archTI,
	}
}

// Init initializes the Bubble Tea model and returns initial commands.
// Starts the periodic tick and begins listening for device events.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),              // Start periodic refresh
		listenForEvents(m.manager), // Start listening for device events
	)
}

// Update handles incoming messages and updates the model state.
// This is the core Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Delegate keyboard input to handleKey
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		// Track terminal dimensions for responsive layout
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Periodic refresh: update device list from manager
		m.devices = m.manager.Devices()
		m.sortDevices()
		m.clampCursor()
		return m, tickCmd() // Schedule next tick

	case deviceEventMsg:
		// Device event: refresh list and re-subscribe
		m.devices = m.manager.Devices()
		m.sortDevices()
		m.clampCursor()
		return m, listenForEvents(m.manager)
	}

	// Handle text input when connect dialog is open
	if m.showConnect {
		var cmd tea.Cmd
		m.connectInput, cmd = m.connectInput.Update(msg)
		return m, cmd
	}

	// Handle text input when rename dialog is open
	if m.showRename {
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}

	// Handle text input when arch dialog is open
	if m.showArch {
		var cmd tea.Cmd
		m.archInput, cmd = m.archInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey processes keyboard input based on current mode.
// In dialog modes, handles text input.
// In normal mode, handles navigation and actions.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Connect dialog mode: handle address input
	if m.showConnect {
		switch msg.String() {
		case "enter":
			// Submit: initiate connection to entered address
			addr := m.connectInput.Value()
			if addr != "" {
				go m.manager.Connect(addr) // Connect in background
			}
			m.showConnect = false
			m.connectInput.SetValue("")
			return m, nil
		case "esc":
			// Cancel: close dialog without connecting
			m.showConnect = false
			m.connectInput.SetValue("")
			return m, nil
		default:
			// Pass other keys to text input for typing
			var cmd tea.Cmd
			m.connectInput, cmd = m.connectInput.Update(msg)
			return m, cmd
		}
	}

	// Rename dialog mode: handle name input
	if m.showRename {
		switch msg.String() {
		case "enter":
			// Submit: set custom name on selected device
			name := m.renameInput.Value()
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				device := m.devices[m.cursor]
				device.SetCustomName(name) // Empty string clears custom name
			}
			m.showRename = false
			m.renameInput.SetValue("")
			return m, nil
		case "esc":
			// Cancel: close dialog without renaming
			m.showRename = false
			m.renameInput.SetValue("")
			return m, nil
		default:
			// Pass other keys to text input for typing
			var cmd tea.Cmd
			m.renameInput, cmd = m.renameInput.Update(msg)
			return m, cmd
		}
	}

	// Arch dialog mode: handle architecture input
	if m.showArch {
		switch msg.String() {
		case "enter":
			// Submit: set custom arch on selected device
			arch := m.archInput.Value()
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				device := m.devices[m.cursor]
				device.SetCustomArch(arch) // Empty string clears custom arch
			}
			m.showArch = false
			m.archInput.SetValue("")
			return m, nil
		case "esc":
			// Cancel: close dialog without changing arch
			m.showArch = false
			m.archInput.SetValue("")
			return m, nil
		default:
			// Pass other keys to text input for typing
			var cmd tea.Cmd
			m.archInput, cmd = m.archInput.Update(msg)
			return m, cmd
		}
	}

	// Color picker mode: handle color selection
	if m.showColor {
		switch msg.String() {
		case "enter":
			// Submit: set selected color on device
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				device := m.devices[m.cursor]
				color := colorOptions[m.colorCursor].hex
				device.SetCustomColor(color)
			}
			m.showColor = false
			return m, nil
		case "esc":
			// Cancel: close picker without changing color
			m.showColor = false
			return m, nil
		case "left", "h":
			// Move cursor left in color picker
			if m.colorCursor > 0 {
				m.colorCursor--
			}
			return m, nil
		case "right", "l":
			// Move cursor right in color picker
			if m.colorCursor < len(colorOptions)-1 {
				m.colorCursor++
			}
			return m, nil
		}
		return m, nil
	}

	// Normal mode: handle navigation and actions
	switch msg.String() {
	case "q", "ctrl+c":
		// Quit: exit TUI without selecting a device
		m.quit = true
		return m, tea.Quit

	case "up", "k":
		// Move cursor up
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		// Move cursor down
		if m.cursor < len(m.devices)-1 {
			m.cursor++
		}

	case "enter":
		// Select: set selected device and exit TUI (or reconnect if disconnected)
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			device := m.devices[m.cursor]
			if device.GetState() == connection.DeviceDisconnected {
				// Attempt to reconnect in background
				go m.manager.ReconnectDevice(device.ID)
				return m, nil
			}
			m.selected = device
			return m, tea.Quit
		}

	case "c":
		// Connect: open connect dialog
		m.showConnect = true
		m.connectInput.Focus()
		return m, textinput.Blink // Start cursor blink

	case "d":
		// Disconnect/Remove: disconnect if live, remove permanently if dead
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			device := m.devices[m.cursor]
			if device.GetState() == connection.DeviceDisconnected {
				// Already dead - remove permanently
				m.manager.RemoveDevicePermanently(device.ID)
			} else {
				// Disconnect (keeps in list as dead)
				m.manager.Disconnect(device.ID)
			}
		}

	case "n":
		// Rename: open rename dialog for selected device
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			device := m.devices[m.cursor]
			// Pre-fill with current custom name (if any)
			m.renameInput.SetValue(device.GetCustomName())
			m.showRename = true
			m.renameInput.Focus()
			return m, textinput.Blink // Start cursor blink
		}

	case "a":
		// Set arch: open arch dialog for selected device
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			device := m.devices[m.cursor]
			// Pre-fill with current custom arch (if any)
			m.archInput.SetValue(device.GetCustomArch())
			m.showArch = true
			m.archInput.Focus()
			return m, textinput.Blink // Start cursor blink
		}

	case "h":
		// Highlight: open color picker for selected device
		if len(m.devices) > 0 && m.cursor < len(m.devices) {
			device := m.devices[m.cursor]
			// Find current color in options, default to purple (index 0)
			m.colorCursor = 0
			currentColor := device.GetCustomColor()
			for i, opt := range colorOptions {
				if opt.hex == currentColor {
					m.colorCursor = i
					break
				}
			}
			m.showColor = true
			return m, nil
		}
	}

	return m, nil
}

// sortDevices sorts the device list by connection time (oldest first).
// This provides a stable ordering - devices don't jump around when names change.
// New devices appear at the bottom of the list.
func (m *Model) sortDevices() {
	sort.Slice(m.devices, func(i, j int) bool {
		return m.devices[i].ConnectedAt.Before(m.devices[j].ConnectedAt)
	})
}

// clampCursor ensures the cursor stays within valid bounds.
// Called after any change to the device list.
func (m *Model) clampCursor() {
	if m.cursor >= len(m.devices) {
		m.cursor = len(m.devices) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// Selected returns the device selected by the user.
// Returns nil if the user quit without selecting.
func (m Model) Selected() *connection.Device {
	if m.quit {
		return nil
	}
	return m.selected
}

// tickCmd returns a Bubble Tea command that sends a tickMsg after 1 second.
// Used for periodic UI refresh.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// listenForEvents returns a Bubble Tea command that waits for device events.
// When an event arrives, it's wrapped in deviceEventMsg and sent to Update.
func listenForEvents(manager *connection.Manager) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-manager.Events()
		if !ok {
			return nil // Channel closed
		}
		return deviceEventMsg(event)
	}
}

// RunDeviceList runs the Bubble Tea device list and returns the selected device.
// This is the main entry point called from main.go.
// Returns nil if the user quit without selecting a device.
func RunDeviceList(manager *connection.Manager) *connection.Device {
	model := NewModel(manager)
	p := tea.NewProgram(model, tea.WithAltScreen()) // Use alternate screen buffer

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running device list: %v\n", err)
		return nil
	}

	return finalModel.(Model).Selected()
}
