package commands

// Transfer commands for uploading and downloading files to/from the remote device.
// These commands handle large file transfers with progress tracking.

import (
	"fmt"
	"os"

	"github.com/Necromancer-Labs/embbridge-tui/internal/connection"
	"github.com/spf13/cobra"
)

// PullCmd downloads a file from the remote device to the local filesystem.
// Usage: pull <remote> [local]
// If local path is not specified, uses the remote filename.
// Shows progress during transfer and tracks transfer state in device.
func (m *Module) PullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <remote> [local]",
		Short: "Download file from device",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			device := m.GetDevice()
			if session == nil {
				PrintError("No active session")
				return
			}

			remotePath := args[0]
			localPath := remotePath // Default to same filename
			if len(args) > 1 {
				localPath = args[1]
			}

			// Start transfer tracking (updates device state to Transferring)
			if device != nil {
				device.StartTransfer(connection.TransferPull, remotePath, localPath, 0)
				defer device.EndTransfer()
			}

			// Pull file with progress callback
			data, size, mode, err := session.Pull(remotePath, func(transferred, total int64) {
				// Update device transfer progress for TUI display
				if device != nil {
					device.UpdateTransferProgress(transferred)
				}
				// Print progress to terminal
				pct := float64(transferred) / float64(total) * 100
				fmt.Printf("\rDownloading: %.1f%% (%d/%d bytes)", pct, transferred, total)
			})
			fmt.Println() // Newline after progress

			if err != nil {
				PrintError(err.Error())
				return
			}

			// Write downloaded data to local file with original permissions
			if err := os.WriteFile(localPath, data, os.FileMode(mode)); err != nil {
				PrintError("Failed to write file: " + err.Error())
				return
			}

			PrintSuccess(fmt.Sprintf("Downloaded: %s (%d bytes)", localPath, size))
		},
	}
}

// PushCmd uploads a file from the local filesystem to the remote device.
// Usage: push <local> [remote]
// If remote path is not specified, uses the local filename.
// Shows progress during transfer and tracks transfer state in device.
func (m *Module) PushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push <local> [remote]",
		Short: "Upload file to device",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			device := m.GetDevice()
			if session == nil {
				PrintError("No active session")
				return
			}

			localPath := args[0]
			remotePath := localPath // Default to same filename
			if len(args) > 1 {
				remotePath = args[1]
			}

			// Read local file into memory
			data, err := os.ReadFile(localPath)
			if err != nil {
				PrintError("Failed to read file: " + err.Error())
				return
			}

			// Get file permissions to preserve on remote
			info, err := os.Stat(localPath)
			if err != nil {
				PrintError("Failed to stat file: " + err.Error())
				return
			}

			// Start transfer tracking (updates device state to Transferring)
			if device != nil {
				device.StartTransfer(connection.TransferPush, remotePath, localPath, int64(len(data)))
				defer device.EndTransfer()
			}

			// Push file with progress callback
			err = session.Push(remotePath, data, uint32(info.Mode()), func(transferred, total int64) {
				// Update device transfer progress for TUI display
				if device != nil {
					device.UpdateTransferProgress(transferred)
				}
				// Print progress to terminal
				pct := float64(transferred) / float64(total) * 100
				fmt.Printf("\rUploading: %.1f%% (%d/%d bytes)", pct, transferred, total)
			})
			fmt.Println() // Newline after progress

			if err != nil {
				PrintError(err.Error())
				return
			}

			PrintSuccess(fmt.Sprintf("Uploaded: %s (%d bytes)", remotePath, len(data)))
		},
	}
}
