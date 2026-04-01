package commands

// Network commands for querying network configuration on the remote device.
// These commands provide information about network interfaces, routing,
// and port forwarding tunnels.

import (
	"fmt"
	"strconv"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
	"github.com/spf13/cobra"
)

// activeTunnel holds the currently active port forward tunnel
var activeTunnel *protocol.ForwardTunnel

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

// ForwardCmd opens a TCP port forward tunnel through the agent.
// Usage: forward-tcp <localport> <remotehost> <remoteport>
//
// This allows accessing services on the device's network from your local machine.
// The tunnel flows: localhost:localport -> agent -> remotehost:remoteport
//
// ForwardOpen/ForwardClose go through the session mutex to avoid racing
// with the heartbeat goroutine that also reads from the socket.
//
// Examples:
//
//	forward-tcp 8080 192.168.1.100 80   # Access neighbor device's web UI
//	forward-tcp 8554 localhost 554     # Access device's RTSP stream
//	forward-tcp 9000 10.0.0.1 22       # SSH to another device
func (m *Module) ForwardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "forward-tcp <localport> <remotehost> <remoteport>",
		Short: "Open a TCP port forward tunnel through the agent",
		Long: `Forward local TCP port to a remote host via the agent.

Examples:
  forward-tcp 8080 192.168.1.100 80   Access neighbor device's web UI at localhost:8080
  forward-tcp 8554 localhost 554     Access device's RTSP stream at localhost:8554
  forward-tcp 9000 10.0.0.1 22       SSH to another device via localhost:9000`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			// Parse local port
			localPort, err := strconv.ParseUint(args[0], 10, 16)
			if err != nil || localPort == 0 || localPort > 65535 {
				PrintError(fmt.Sprintf("Invalid local port: %s", args[0]))
				return
			}

			// Parse remote port
			remotePort, err := strconv.ParseUint(args[2], 10, 16)
			if err != nil || remotePort == 0 || remotePort > 65535 {
				PrintError(fmt.Sprintf("Invalid remote port: %s", args[2]))
				return
			}

			remoteHost := args[1]

			// Close any existing tunnel
			if activeTunnel != nil && activeTunnel.IsRunning() {
				fmt.Println("Closing existing tunnel...")
				activeTunnel.StopForwarding()
				session.ForwardClose()
				activeTunnel.Cleanup()
			}

			localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

			// Open tunnel through session (mutex-protected, safe from heartbeat)
			resp, err := session.ForwardOpen(remoteHost, uint16(remotePort), "tcp")
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Create tunnel and start forwarding with the pre-obtained ID
			activeTunnel = protocol.NewForwardTunnel(session.GetProto(), localAddr, "tcp")

			fmt.Printf("Opening tunnel: localhost:%d -> %s:%d (via agent)\n",
				localPort, remoteHost, remotePort)

			if err := activeTunnel.StartForwarding(resp.ID); err != nil {
				PrintError(err.Error())
				activeTunnel = nil
				return
			}

			PrintSuccess(fmt.Sprintf("Tunnel active! Connect to %s", activeTunnel.LocalAddr()))
			fmt.Println("Use 'forward-stop' to close the tunnel")
		},
	}
}

// ForwardUDPCmd opens a UDP port forward tunnel through the agent.
// Usage: forward-udp <localport> <remotehost> <remoteport>
//
// Same architecture as ForwardCmd but uses UDP transport.
// Useful for inspecting UPnP/SSDP and other UDP services on the LAN.
func (m *Module) ForwardUDPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "forward-udp <localport> <remotehost> <remoteport>",
		Short: "Open a UDP port forward tunnel through the agent",
		Long: `Forward local UDP port to a remote host via the agent.

Examples:
  forward-udp 1900 239.255.255.250 1900   Inspect UPnP/SSDP on the LAN
  forward-udp 5353 224.0.0.251 5353        Inspect mDNS on the LAN`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			localPort, err := strconv.ParseUint(args[0], 10, 16)
			if err != nil || localPort == 0 || localPort > 65535 {
				PrintError(fmt.Sprintf("Invalid local port: %s", args[0]))
				return
			}

			remotePort, err := strconv.ParseUint(args[2], 10, 16)
			if err != nil || remotePort == 0 || remotePort > 65535 {
				PrintError(fmt.Sprintf("Invalid remote port: %s", args[2]))
				return
			}

			remoteHost := args[1]

			// Close any existing tunnel
			if activeTunnel != nil && activeTunnel.IsRunning() {
				fmt.Println("Closing existing tunnel...")
				activeTunnel.StopForwarding()
				session.ForwardClose()
				activeTunnel.Cleanup()
			}

			localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

			// Open UDP tunnel through session (mutex-protected)
			resp, err := session.ForwardOpen(remoteHost, uint16(remotePort), "udp")
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Create UDP tunnel and start forwarding
			activeTunnel = protocol.NewForwardTunnel(session.GetProto(), localAddr, "udp")

			fmt.Printf("Opening UDP tunnel: localhost:%d -> %s:%d (via agent)\n",
				localPort, remoteHost, remotePort)

			if err := activeTunnel.StartForwarding(resp.ID); err != nil {
				PrintError(err.Error())
				activeTunnel = nil
				return
			}

			PrintSuccess(fmt.Sprintf("UDP tunnel active! Send datagrams to %s", activeTunnel.LocalAddr()))
			fmt.Println("Use 'forward-stop' to close the tunnel")
		},
	}
}

// ForwardStopCmd closes the active port forward tunnel.
// Usage: forward-stop
func (m *Module) ForwardStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "forward-stop",
		Short: "Close the active port forward tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			if activeTunnel == nil || !activeTunnel.IsRunning() {
				PrintError("No active tunnel")
				return
			}

			session := m.GetSession()

			// Stop forwarding (keep dispatcher alive for close handshake)
			activeTunnel.StopForwarding()

			// Send close through session (mutex-protected, dispatcher routes response)
			if session != nil {
				session.ForwardClose()
			}

			// Stop dispatcher and wait for goroutines
			activeTunnel.Cleanup()
			activeTunnel = nil
			PrintSuccess("Tunnel closed")
		},
	}
}

// ForwardStatusCmd shows the status of the active tunnel.
// Usage: forward-status
func (m *Module) ForwardStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "forward-status",
		Short: "Show the status of the active port forward tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			if activeTunnel == nil || !activeTunnel.IsRunning() {
				fmt.Println("No active tunnel")
				return
			}

			PrintSuccess(fmt.Sprintf("Tunnel active: %s (%s)", activeTunnel.LocalAddr(), activeTunnel.Transport()))
		},
	}
}
