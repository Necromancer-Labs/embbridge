/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Port forward commands: tunnel TCP/UDP connections through the agent
 *
 * Usage:
 *   forward-tcp <localport> <remotehost> <remoteport>
 *   forward-udp <localport> <remotehost> <remoteport>
 *
 * Example:
 *   forward-tcp 8080 192.168.1.100 80   # Access device's neighbor at http://localhost:8080
 *   forward-udp 1900 239.255.255.250 1900  # Inspect UPnP/SSDP on the LAN
 */

package shell

import (
	"fmt"
	"strconv"

	"github.com/Necromancer-Labs/embbridge/client/protocol"
)

// activeTunnel holds the currently active tunnel (single-stream support)
var activeTunnel *protocol.ForwardTunnel

// doForward opens a port forward tunnel
func (m *EDBModule) doForward(localPort uint16, remoteHost string, remotePort uint16) {
	// Close any existing tunnel
	if activeTunnel != nil && activeTunnel.IsRunning() {
		fmt.Println("Closing existing tunnel...")
		activeTunnel.Stop()
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

	// Create new tunnel
	activeTunnel = protocol.NewForwardTunnel(m.proto, localAddr, "tcp")

	fmt.Printf("Opening tunnel: localhost:%d -> %s:%d (via agent)\n",
		localPort, remoteHost, remotePort)

	if err := activeTunnel.Start(remoteHost, remotePort); err != nil {
		fmt.Printf("Error: %v\n", err)
		activeTunnel = nil
		return
	}

	fmt.Printf("Tunnel active! Connect to %s\n", activeTunnel.LocalAddr())
	fmt.Println("Use 'forward-stop' to close the tunnel")
}

// doForwardStop closes the active tunnel
func (m *EDBModule) doForwardStop() {
	if activeTunnel == nil || !activeTunnel.IsRunning() {
		fmt.Println("No active tunnel")
		return
	}

	activeTunnel.Stop()
	activeTunnel = nil
	fmt.Println("Tunnel closed")
}

// doForwardStatus shows the status of the active tunnel
func (m *EDBModule) doForwardStatus() {
	if activeTunnel == nil || !activeTunnel.IsRunning() {
		fmt.Println("No active tunnel")
		return
	}

	fmt.Printf("Tunnel active: %s (%s)\n", activeTunnel.LocalAddr(), activeTunnel.Transport())
}

// doForwardUDP opens a UDP port forward tunnel
func (m *EDBModule) doForwardUDP(localPort uint16, remoteHost string, remotePort uint16) {
	// Close any existing tunnel
	if activeTunnel != nil && activeTunnel.IsRunning() {
		fmt.Println("Closing existing tunnel...")
		activeTunnel.Stop()
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

	// Create new UDP tunnel
	activeTunnel = protocol.NewForwardTunnel(m.proto, localAddr, "udp")

	fmt.Printf("Opening UDP tunnel: localhost:%d -> %s:%d (via agent)\n",
		localPort, remoteHost, remotePort)

	if err := activeTunnel.Start(remoteHost, remotePort); err != nil {
		fmt.Printf("Error: %v\n", err)
		activeTunnel = nil
		return
	}

	fmt.Printf("UDP tunnel active! Send datagrams to %s\n", activeTunnel.LocalAddr())
	fmt.Println("Use 'forward-stop' to close the tunnel")
}

// parsePort parses a port string and validates the range
func parsePort(s string) (uint16, error) {
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", s)
	}
	if port == 0 || port > 65535 {
		return 0, fmt.Errorf("port out of range: %d", port)
	}
	return uint16(port), nil
}
