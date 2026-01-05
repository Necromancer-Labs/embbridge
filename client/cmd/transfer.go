/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File transfer commands - get and put
 */

package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <device> <remote-path> [local-path]",
	Short: "Download a file from the device",
	Long: `Download a file from the device to local storage.

Examples:
  edb get 192.168.1.50 /etc/passwd ./passwd
  edb get 192.168.1.50 /etc/passwd         # Saves to current directory`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runGet,
}

var putCmd = &cobra.Command{
	Use:   "put <device> <local-path> <remote-path>",
	Short: "Upload a file to the device",
	Long: `Upload a file from local storage to the device.

Examples:
  edb put 192.168.1.50 ./script.sh /tmp/script.sh`,
	Args: cobra.ExactArgs(3),
	RunE: runPut,
}

func runGet(cmd *cobra.Command, args []string) error {
	// TODO: Implement file download
	return nil
}

func runPut(cmd *cobra.Command, args []string) error {
	// TODO: Implement file upload
	return nil
}
