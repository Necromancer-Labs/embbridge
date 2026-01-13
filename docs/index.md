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
| [Architecture Reference](architecture-reference.md) | Device architecture identification guide |
| [Building](building.md) | Build from source |
| [Protocol](protocol.md) | Wire protocol specification |

## Supported Architectures

| Agent Build            | Architecture                 | Typical Devices / Good Examples                                                                                                                                                                                                      |
| ---------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **`edb-agent-arm`**    | **ARM 32-bit (ARMv5/ARMv7)** | • Smart TVs (many Samsung, LG, Sony non-Android models) <br> • Older Android TVs (32-bit builds) <br> • Classic Raspberry Pi models (Pi 1 / Pi Zero) <br> • Many consumer IoT cameras (TP-Link, Wyze v2) <br> • Legacy set-top boxes |
| **`edb-agent-arm64`**  | **ARM 64-bit (AArch64)**     | • Modern Smart TVs (Android TV / Google TV, newer Tizen/WebOS) <br> • Raspberry Pi 3/4/5 <br> • Modern Android phones/tablets <br> • ARM servers (Ampere, AWS Graviton) <br> • New NVR / IoT edge devices                            |
| **`edb-agent-mips`**   | **MIPS32 Big-Endian**        | • Older Cisco/Juniper enterprise routers <br> • Some legacy IP phones & DSP boards <br> • Very early enterprise networking appliances                                                                                                |
| **`edb-agent-mipsel`** | **MIPS32 Little-Endian**     | • Consumer Wi-Fi routers (e.g., **D-Link DIR-645**, Linksys WRT54GL variants) <br> • Atheros/Qualcomm-Atheros based SOHO APs <br> • Older NAS devices <br> • Many early OpenWrt hardware targets                                     |


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
