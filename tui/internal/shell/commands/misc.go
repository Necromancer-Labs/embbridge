package commands

// Miscellaneous commands for shell execution, string extraction, and device control.

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// ExecCmd executes a shell command on the remote device.
// Usage: exec <command> [args...]
// Runs the command through the device's shell and returns stdout/stderr.
// This is useful for running commands not directly supported by embbridge.
func (m *Module) ExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <command>",
		Short: "Execute a shell command",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			// Join all args to form the complete command string
			command := ""
			for i, arg := range args {
				if i > 0 {
					command += " "
				}
				command += arg
			}

			resp, err := session.Exec(command)
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Print stdout (agent sends as binary)
			switch stdout := resp.Data["stdout"].(type) {
			case []byte:
				if len(stdout) > 0 {
					fmt.Print(string(stdout))
				}
			case string:
				if stdout != "" {
					fmt.Print(stdout)
				}
			}

			// Print stderr (agent sends as binary)
			switch stderr := resp.Data["stderr"].(type) {
			case []byte:
				if len(stderr) > 0 {
					fmt.Print(string(stderr))
				}
			case string:
				if stderr != "" {
					fmt.Print(stderr)
				}
			}
		},
	}
}

// StringsCmd extracts printable strings from a file on the remote device.
// Usage: strings [-n minlen] <file>
// Similar to the Unix strings utility, useful for analyzing binaries.
func (m *Module) StringsCmd() *cobra.Command {
	var minLen int
	cmd := &cobra.Command{
		Use:   "strings <file>",
		Short: "Extract printable strings from file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Strings(args[0], minLen)
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Agent sends "content" as binary (newline-separated strings)
			switch content := resp.Data["content"].(type) {
			case []byte:
				fmt.Print(string(content))
			case string:
				fmt.Print(content)
			}
		},
	}
	cmd.Flags().IntVarP(&minLen, "min", "n", 4, "Minimum string length")
	return cmd
}

// KillCmd sends a signal to a process on the remote device.
// Usage: kill <pid> [signal]
// Default signal is 9 (SIGKILL).
func (m *Module) KillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kill <pid> [signal]",
		Short: "Send a signal to a process (default: 9/SIGKILL)",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			pid, err := strconv.Atoi(args[0])
			if err != nil {
				PrintError(fmt.Sprintf("Invalid pid: %s", args[0]))
				return
			}

			sig := 0
			if len(args) > 1 {
				sig, err = strconv.Atoi(args[1])
				if err != nil {
					PrintError(fmt.Sprintf("Invalid signal: %s", args[1]))
					return
				}
			}

			resp, err := session.Kill(pid, sig)
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			if sig == 0 {
				sig = 9
			}
			PrintSuccess(fmt.Sprintf("Sent signal %d to pid %d", sig, pid))
		},
	}
}

// RebootCmd reboots the remote device.
// Usage: reboot
// Sends a reboot command to the device. The connection will be lost after this.
func (m *Module) RebootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reboot",
		Short: "Reboot the device",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Reboot()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess("Device is rebooting...")
		},
	}
}
