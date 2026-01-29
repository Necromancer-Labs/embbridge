package commands

// Filesystem commands for interacting with the remote device's filesystem.
// These commands mirror standard Unix filesystem utilities.

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// LsCmd lists directory contents on the remote device.
// Usage: ls [path]
// If no path is provided, lists the current directory.
func (m *Module) LsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory contents",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			// Default to current directory if no path specified
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			resp, err := session.Ls(path)
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatLsOutput(resp.Data))
		},
	}
}

// CdCmd changes the current directory on the remote device.
// Usage: cd <path>
// Updates the shell prompt to show the new directory.
func (m *Module) CdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cd <path>",
		Short: "Change directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Cd(args[0])
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Update prompt to show new directory
			m.UpdatePrompt()
		},
	}
}

// PwdCmd prints the current working directory on the remote device.
// Usage: pwd
func (m *Module) PwdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pwd",
		Short: "Print current directory",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Pwd()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			if path, ok := resp.Data["path"].(string); ok {
				fmt.Println(path)
			}
		},
	}
}

// CatCmd displays the contents of a file on the remote device.
// Usage: cat <file>
// Handles both binary and string content from the agent.
func (m *Module) CatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat <file>",
		Short: "Display file contents",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Cat(args[0])
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Content may be []byte (binary) or string depending on msgpack decoding
			switch content := resp.Data["content"].(type) {
			case []byte:
				fmt.Print(string(content))
			case string:
				fmt.Print(content)
			}
		},
	}
}

// RmCmd removes a file or empty directory on the remote device.
// Usage: rm <path>
func (m *Module) RmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <path>",
		Short: "Remove file or empty directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Rm(args[0])
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess("Removed: " + args[0])
		},
	}
}

// MvCmd moves or renames a file or directory on the remote device.
// Usage: mv <src> <dst>
func (m *Module) MvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <src> <dst>",
		Short: "Move/rename file or directory",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Mv(args[0], args[1])
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess(fmt.Sprintf("Moved: %s -> %s", args[0], args[1]))
		},
	}
}

// CpCmd copies a file on the remote device.
// Usage: cp <src> <dst>
func (m *Module) CpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cp <src> <dst>",
		Short: "Copy file",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Cp(args[0], args[1])
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess(fmt.Sprintf("Copied: %s -> %s", args[0], args[1]))
		},
	}
}

// MkdirCmd creates a directory on the remote device.
// Usage: mkdir [-m mode] <path>
// Default mode is 0755 if not specified.
func (m *Module) MkdirCmd() *cobra.Command {
	var mode uint32 = 0755
	cmd := &cobra.Command{
		Use:   "mkdir <path>",
		Short: "Create directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Mkdir(args[0], mode)
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess("Created: " + args[0])
		},
	}
	cmd.Flags().Uint32VarP(&mode, "mode", "m", 0755, "Directory permissions")
	return cmd
}

// ChmodCmd changes file permissions on the remote device.
// Usage: chmod <mode> <path>
// Mode is specified in octal (e.g., 755, 644).
func (m *Module) ChmodCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chmod <mode> <path>",
		Short: "Change file permissions",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			// Parse mode as octal (base 8)
			mode, err := strconv.ParseUint(args[0], 8, 32)
			if err != nil {
				PrintError("Invalid mode: " + args[0])
				return
			}

			resp, err := session.Chmod(args[1], uint32(mode))
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			PrintSuccess(fmt.Sprintf("Changed mode of %s to %o", args[1], mode))
		},
	}
}
