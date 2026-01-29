package commands

// System commands for querying device state and system information.
// These commands provide information about processes, network connections,
// system identity, kernel logs, and hardware.

import (
	"fmt"

	"github.com/spf13/cobra"
)

// PsCmd lists running processes on the remote device.
// Usage: ps
// Displays PID, PPID, state, and command line for each process.
func (m *Module) PsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List processes",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Ps()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatPsOutput(resp.Data))
		},
	}
}

// SsCmd lists network connections on the remote device.
// Usage: ss
// Displays protocol, state, local/remote addresses, and associated process.
func (m *Module) SsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ss",
		Short: "List network connections",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Ss()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatSsOutput(resp.Data))
		},
	}
}

// UnameCmd shows system information from the remote device.
// Usage: uname
// Displays system name, hostname, kernel version, and architecture.
func (m *Module) UnameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uname",
		Short: "Show system information",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Uname()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatUnameOutput(resp.Data))
		},
	}
}

// WhoamiCmd shows the current user on the remote device.
// Usage: whoami
// Displays username, UID, and GID.
func (m *Module) WhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current user",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Whoami()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatWhoamiOutput(resp.Data))
		},
	}
}

// DmesgCmd shows the kernel log from the remote device.
// Usage: dmesg
// Displays the contents of the kernel ring buffer.
func (m *Module) DmesgCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dmesg",
		Short: "Show kernel log",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Dmesg()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			// Agent sends "log" as binary (raw kernel log output)
			switch log := resp.Data["log"].(type) {
			case []byte:
				fmt.Print(string(log))
			case string:
				fmt.Print(log)
			}
		},
	}
}

// CpuinfoCmd shows CPU information from the remote device.
// Usage: cpuinfo
// Displays the raw contents of /proc/cpuinfo.
func (m *Module) CpuinfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cpuinfo",
		Short: "Show CPU information",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Cpuinfo()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatCpuinfoOutput(resp.Data))
		},
	}
}

// MtdCmd lists MTD (Memory Technology Device) partitions on the remote device.
// Usage: mtd
// Displays flash memory partitions, commonly found on embedded devices.
func (m *Module) MtdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mtd",
		Short: "List MTD partitions",
		Run: func(cmd *cobra.Command, args []string) {
			session := m.GetSession()
			if session == nil {
				PrintError("No active session")
				return
			}

			resp, err := session.Mtd()
			if err != nil {
				PrintError(err.Error())
				return
			}
			if !resp.OK {
				PrintError(resp.Error)
				return
			}

			fmt.Print(FormatMtdOutput(resp.Data))
		},
	}
}
