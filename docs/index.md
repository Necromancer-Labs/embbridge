# embbridge

Embedded Debug Bridge - adb for embedded Linux systems.

## Overview

embbridge is a lightweight agent/client tool for interacting with embedded Linux devices. It provides a consistent interface for firmware analysis, security research, and device forensics.

## Key Features

- **Single static binary** - No dependencies, runs on any Linux system
- **Small footprint** - Agent binaries are 100-200KB
- **Cross-platform** - Supports ARM, ARM64, MIPS, MIPSEL, x86_64
- **File transfer** - Pull and push files with progress indication
- **System inspection** - Process listing, network connections, kernel logs
- **No shell required** - Works without busybox or any shell on target

## Use Cases

- Firmware extraction and analysis
- Embedded device forensics
- Security research and vulnerability assessment
- IoT device debugging
- Router and access point analysis

## Documentation

| Document | Description |
|----------|-------------|
| [Quick Start](quickstart.md) | Get running in 5 minutes |
| [Commands](commands.md) | Complete command reference |
| [Building](building.md) | Build from source |
| [Protocol](protocol.md) | Wire protocol specification |

## Supported Architectures

| Binary | Architecture | Typical Devices |
|--------|--------------|-----------------|
| `edb-agent-arm` | ARM 32-bit (ARMv5+) | Raspberry Pi, routers, IoT devices |
| `edb-agent-arm64` | ARM 64-bit (AArch64) | Modern ARM servers and SBCs |
| `edb-agent-mips` | MIPS32 big-endian | Broadcom/Atheros routers |
| `edb-agent-mipsel` | MIPS32 little-endian | Consumer routers, embedded devices |
| `edb-agent-x86_64` | x86 64-bit | Standard Linux systems, VMs |

## Client Binary

| Binary | Platform |
|--------|----------|
| `edb-client-linux-amd64` | Linux x86_64 |

## Quick Example

```bash
# On target device (agent)
./edb-agent -l 1337

# On your workstation (client)
./edb shell 192.168.1.50:1337

# Now you have a shell
edb[/]# ls /etc
edb[/]# pull /etc/passwd ./passwd.txt
edb[/]# ps
edb[/]# ss
```

## License

MIT License - [Necromancer Labs](https://github.com/Necromancer-Labs)

## Links

- [GitHub Repository](https://github.com/Necromancer-Labs/embbridge)
- [Releases](https://github.com/Necromancer-Labs/embbridge/releases)
