# embbridge

**Embedded Debug Bridge** — adb, but for embedded systems.

A lightweight agent/client tool for interacting with embedded devices. Useful for firmware analysis, security research, and device forensics.

## The Problem

Android has `adb`. It's standard, reliable, and works the same on every device. You can always `adb pull`, `adb shell`, `adb push`, etc.

Embedded Linux has nothing like this. Every device is different:
- Different shells (or none)
- Different available utilities (busybox, toybox, UART, telnet)
- Different file transfer options (tftp? scp? wget? none?)
- No consistent way to collect firmware, logs, or artifacts

You never know what you're going to get, this is frustrating. 

## The Solution

embbridge provides an adb-like experience for any embedded Linux system:
- **One agent binary** — statically linked, ~50-180KB, runs anywhere
- **Consistent interface** — same commands work on every device
- **Self-contained** — doesn't rely on target utilities; all commands implemented natively

## Quick Start

**1. Get the agent** — download a release or build from source:
```bash
# Option A: Download pre-built binary from releases
wget https://github.com/Necromancer-Labs/embbridge/releases/latest/download/edb-agent-arm

# Option B: Build from source
cd agent && make arm    # or: make mipsel, make arm64, etc.
```

**2. Transfer agent to device** (use whatever method the device supports):

```bash
# via wget (serve from workstation, fetch from device)
Workstation: python -m http.server 8080
Device:      wget -O /tmp/edb-agent http://192.168.1.100:8080/edb-agent-arm

# via scp
Workstation: scp edb-agent-arm root@192.168.1.50:/tmp/

# via netcat (listen on device, push from workstation)
Device:      nc -l -p 4444 > /tmp/edb-agent
Workstation: nc 192.168.1.50 4444 < edb-agent-arm

# via tftp
Device:      tftp -g -r edb-agent-arm 192.168.1.100

# Then on device:
chmod +x /tmp/edb-agent-arm
/tmp/edb-agent-arm -l 1337
```

**3. Connect from your workstation:**
```bash
./edb shell 192.168.1.50:1337
```

That's it. You now have a consistent interface with file transfer, process listing, and more.

## Features

| Command | Description |
|---------|-------------|
| `ls`, `cd`, `pwd`, `cat` | Directory navigation and file viewing |
| `pull <remote> [local]` | Download file from device |
| `push <local> <remote>` | Upload file to device |
| `rm`, `mv`, `cp`, `mkdir`, `chmod` | File operations |
| `ps` | Process tree |
| `ss` | Network connections with PIDs |
| `uname`, `whoami` | System info |
| `dmesg` | Kernel log |
| `strings <file>` | Extract printable strings |
| `exec <cmd>` | Run binary (no shell) |
| `reboot` | Reboot device |

## Connection Modes

**Bind mode** — agent listens, you connect to it:
```bash
# Device
./edb-agent -l 1337

# Workstation
./edb shell 192.168.1.50:1337
```

**Reverse mode** — agent connects out to you:
```bash
# Workstation
./edb listen

# Device
./edb-agent -c 192.168.1.100:1337
```

## Building

### Client (Go)

```bash
cd client && go build -o edb .
```

### Agent (C)

```bash
cd agent
make              # Native debug build
make release      # Native release build
make arm          # Cross-compile for ARM
make mipsel       # Cross-compile for MIPS little-endian
make all-arch     # Build all architectures
```

Cross-compilation requires buildroot toolchains. Run `./scripts/build-toolchains.sh` first (one-time, ~30 min per arch).

### Supported Architectures

| Binary | Compatibility | Typical Devices |
|--------|---------------|-----------------|
| `edb-agent-arm` | ARMv5+ (v5, v6, v7, v8 32-bit) | Raspberry Pi, routers, IoT |
| `edb-agent-arm64` | AArch64 | Modern ARM systems |
| `edb-agent-mips` | MIPS32 big-endian | Broadcom/Atheros routers |
| `edb-agent-mipsel` | MIPS32 little-endian | Consumer routers |

All binaries are statically linked (~50-180KB) with no runtime dependencies.

## Design

- **Agent (C)**: Tiny static binary. No external dependencies. All commands implemented natively — no reliance on busybox or target shell.
- **Client (Go)**: Interactive readline shell with history.
- **Protocol**: MessagePack over TCP with length-prefixed framing.

## License

MIT

---

*Necromancer Labs*
