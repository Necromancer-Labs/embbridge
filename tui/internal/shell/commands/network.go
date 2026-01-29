package commands

// Network commands for querying network configuration on the remote device.
// These commands provide information about network interfaces and routing.

import (
	"fmt"

	"github.com/spf13/cobra"
)

// IpAddrCmd shows network interfaces on the remote device.
// Usage: ip-addr
// Displays interface names, addresses, and status.
func (m *Module) IpAddrCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip-addr",
		Short: "Show network interfaces",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.IpAddr()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatIpAddrOutput(resp.Data))
		},
	}
}

// IpRouteCmd shows the routing table on the remote device.
// Usage: ip-route
// Displays destination networks, gateways, and interfaces.
func (m *Module) IpRouteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip-route",
		Short: "Show routing table",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.IpRoute()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatIpRouteOutput(resp.Data))
		},
	}
}
