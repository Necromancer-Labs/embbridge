// Package commands provides all device interaction commands for the Graveyard shell.
//
// This package implements gocmd2's CommandModule interface to provide a set of
// cobra commands for interacting with connected embedded devices. Commands are
// organized by category:
//
//   - Filesystem: ls, cd, pwd, cat, rm, mv, cp, mkdir, chmod
//   - System: ps, ss, uname, whoami, dmesg, cpuinfo, mtd
//   - Network: ip-addr, ip-route
//   - Transfer: pull (download), push (upload)
//   - Misc: exec (shell command), strings, reboot
//
// Commands communicate with the device via the embbridge protocol session,
// which is stored in the shell's shared state and retrieved via GetSession().
package commands

import (
	"fmt"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/Necromancer-Labs/embbridge-tui/internal/ui/theme"
	"github.com/Necromancerlabs/gocmd2/pkg/shellapi"
	"github.com/spf13/cobra"
)

// Module implements gocmd2's CommandModule interface.
// It provides all device interaction commands and maintains a reference
// to the shell for accessing shared state (device, session).
type Module struct {
	shell shellapi.ShellAPI // Reference to shell for state access
}

// NewModule creates a new graveyard command module.
// The module is registered with the shell via sh.RegisterModule().
func NewModule() *Module {
	return &Module{}
}

// Name returns the module name used for identification.
func (m *Module) Name() string {
	return "graveyard"
}

// Initialize is called by gocmd2 when the module is registered.
// Stores the shell reference for later state access.
func (m *Module) Initialize(shell shellapi.ShellAPI) {
	m.shell = shell
}

// GetCommands returns all cobra commands provided by this module.
// These commands are added to the shell's root command.
func (m *Module) GetCommands() []*cobra.Command {
	return []*cobra.Command{
		// Filesystem commands (fs.go)
		m.LsCmd(),
		m.CdCmd(),
		m.PwdCmd(),
		m.CatCmd(),
		m.RmCmd(),
		m.MvCmd(),
		m.CpCmd(),
		m.MkdirCmd(),
		m.ChmodCmd(),

		// System commands (system.go)
		m.PsCmd(),
		m.SsCmd(),
		m.UnameCmd(),
		m.WhoamiCmd(),
		m.DmesgCmd(),
		m.CpuinfoCmd(),
		m.MtdCmd(),

		// Network commands (network.go)
		m.IpAddrCmd(),
		m.IpRouteCmd(),

		// Transfer commands (transfer.go)
		m.PullCmd(),
		m.PushCmd(),

		// Misc commands (misc.go)
		m.ExecCmd(),
		m.StringsCmd(),
		m.RebootCmd(),
	}
}

// GetSession retrieves the active device session from shell state.
// Returns nil if no session is active (commands should show "No active session" error).
// The session is set by shell.go when entering shell mode.
func (m *Module) GetSession() *connection.Session {
	if val, ok := m.shell.GetState("session"); ok {
		if session, ok := val.(*connection.Session); ok {
			return session
		}
	}
	return nil
}

// GetDevice retrieves the active device from shell state.
// Returns nil if no device is active.
// The device is set by shell.go when entering shell mode.
func (m *Module) GetDevice() *connection.Device {
	if val, ok := m.shell.GetState("device"); ok {
		if device, ok := val.(*connection.Device); ok {
			return device
		}
	}
	return nil
}

// UpdatePrompt updates the shell prompt to reflect current device state.
// Called after directory changes (cd) to show the new path.
// Format: graveyard[device-name]:/current/path#
func (m *Module) UpdatePrompt() {
	device := m.GetDevice()
	if device == nil {
		return
	}
	name := device.DisplayName()
	dir := device.GetCurrentDir()
	prompt := theme.PromptStyle.Render(fmt.Sprintf("graveyard[%s]:%s#", name, dir))
	m.shell.SetPrompt(prompt)
}
