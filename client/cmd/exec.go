/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Exec command - run a single command on device (non-interactive)
 */

package cmd

import (
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <device> <command>",
	Short: "Execute a command on the device",
	Long: `Execute a single command on the device and print the output.

Examples:
  edb exec 192.168.1.50 "cat /etc/passwd"
  edb exec 192.168.1.50 "ps aux"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runExec,
}

func runExec(cmd *cobra.Command, args []string) error {
	// TODO: Implement non-interactive exec
	// 1. Connect to device
	// 2. Send exec command
	// 3. Print output
	// 4. Disconnect
	return nil
}
