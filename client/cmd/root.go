/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Root command and CLI setup
 */

package cmd

import (
	"github.com/spf13/cobra"
)

const (
	Version     = "0.1.0"
	DefaultPort = 1337
)

var rootCmd = &cobra.Command{
	Use:   "edb",
	Short: "Embedded Debug Bridge - ADB for embedded systems",
	Long: `embbridge (edb) - An ADB-like device bridge for embedded systems.

Shell, file transfer, and verified artifact bundles for forensics,
investigation, and research.

Usage:
  edb listen              Listen for agent connections (reverse mode)
  edb shell <host:port>   Connect to agent (bind mode)`,
	Version: Version,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(listenCmd)
	rootCmd.AddCommand(shellCmd)
}
