# embbridge

**Embedded Debug Bridge** — an ADB-like tool for embedded systems research.

Goal: One easy tool for most embedded systems. 

## The Problem

Embedded device access and investigation is super fragmented:

- Every brand has different shell access (UART, Telnet, ssh, proprietary protocols)
- File transfer usually doesn't exist. You never really know if you'll be able to get files off the easy way or hard way. 
- No standard way to collect firmware dumps and artifacts

Every device is different. This is frustrating.

## The Solution

A standardized tool with two connection modes:

**Reverse (agent connects to you):**
```bash
# On your workstation
edb listen

# On the device
./edb-agent -c 192.168.1.100:1337
```

**Bind (agent listens, you connect to it):**
```bash
# On the device
./edb-agent -l 1337

# On your workstation
edb shell 192.168.1.50:1337
```

Either way, you get an interactive shell with file transfer:

```
edb[/]> ls
drwxr-xr-x    -  root     3 Jan 18:45  bin
drwxr-xr-x    -  root     3 Jan 18:45  etc
drwxr-xr-x    -  root     3 Jan 18:45  lib
...

edb[/]> cd /etc
edb[/etc]> cat passwd
root:x:0:0:root:/root:/bin/sh

edb[/etc]> get passwd ./              # download file
↓ Downloading passwd...
  45 B downloaded in 0s (1.2 MB/s)

edb[/etc]> put script.sh /tmp/        # upload file
↑ Uploading script.sh (1.2 KB)...
  1.2 KB uploaded in 0s (2.4 MB/s)

edb[/etc]> exec "uname -a"            # run commands
Linux router 4.14.90 #1 SMP armv7l GNU/Linux
```

## Architecture

**Reverse mode** (agent connects out):
```
┌─────────────────────┐         ┌─────────────────────┐
│   edb client (Go)   │◄────────│   edb agent (C)     │
│   your workstation  │  msgpack│   tiny, static      │
│                     │         │   on the device     │
│   edb listen :1337  │◄────────│ ./edb-agent -c IP   │
└─────────────────────┘         └─────────────────────┘
```

**Bind mode** (you connect to agent):
```
┌─────────────────────┐         ┌─────────────────────┐
│   edb client (Go)   │────────►│   edb agent (C)     │
│   your workstation  │  msgpack│   tiny, static      │
│                     │         │   on the device     │
│   edb shell IP:1337 │────────►│ ./edb-agent -l 1337 │
└─────────────────────┘         └─────────────────────┘
```

### Agent (C)
- **Tiny**: Target <100KB static binary
- **No dependencies**: Static linking, runs on anything
- **Multi-arch**: ARM, MIPS, x86, RISC-V
- **Self-contained**: All commands implemented natively - no reliance on target shell (busybox, ash, etc)

### Client (Go)
- **Interactive shell**: Readline-based with history and completion
- **File transfer**: Get and put with progress display
- **Modular**: Built on [gocmd2](https://github.com/Necromancer-Labs/gocmd2)

### Protocol
- **MessagePack over TCP**: Compact binary serialization
- **Simple framing**: 4-byte length prefix + payload
- **Chunked transfers**: Large files transferred in 64KB chunks

## Tech Stack

### Client (Go)
| Component | Purpose | Link |
|-----------|---------|------|
| [gocmd2](https://github.com/Necromancer-Labs/gocmd2) | Interactive shell framework | Modular command shell with readline |
| [cobra](https://github.com/spf13/cobra) | CLI framework | Powers `edb listen`, `edb shell` commands |
| [msgpack](https://github.com/vmihailenco/msgpack) | Serialization | Encodes/decodes protocol messages |

### Agent (C)
| Component | Purpose | Notes |
|-----------|---------|-------|
| Hand-rolled MessagePack | Serialization | Minimal subset, no external deps |
| POSIX syscalls | File operations | `opendir`, `stat`, `fopen`, etc. |
| BSD sockets | Networking | `connect`, `accept`, `send`, `recv` |

No external libraries in the agent - everything is implemented from scratch to keep the binary tiny and portable.

## Current Commands

| Command | Description | Status |
|---------|-------------|--------|
| `ls [path]` | List directory | ✅ |
| `cd <path>` | Change directory | ✅ |
| `pwd` | Print working directory | ✅ |
| `cat <file>` | Print file contents | ✅ |
| `get <remote> [local]` | Download file | ✅ |
| `put <local> <remote>` | Upload file | ✅ |
| `uname` | System information | ✅ |
| `exec <command>` | Run command (no shell) | 🚧 |
| `rm`, `mv`, `cp`, `mkdir`, `chmod` | File operations | 🚧 |
| `mtd`, `firmware` | Firmware dumping | 📋 |
| `bundle` | Artifact collection | 📋 |

## Feature Phases

### Phase 0: Foundation ✅
- [x] Protocol spec (MessagePack framing)
- [x] Agent skeleton (C) with TCP connect/listen
- [x] Client skeleton (Go) with CLI
- [x] Handshake and basic communication
- [x] Navigation commands (ls, cd, pwd, cat)

### Phase 1: Core 🚧
- [x] Interactive shell with readline
- [x] File transfer (get/put) with progress
- [x] Command execution (exec, uname)
- [ ] File operations (rm, mv, cp, mkdir, chmod)
- [ ] Cross-compilation for ARM, MIPS

### Phase 2: Future 📋
- [ ] Firmware dumping (MTD, UBI, block devices)
- [ ] Artifact bundles with profiles
- [ ] Hash manifest generation

### Phase 3: Polish 📋
- [ ] Serial transport (UART)
- [ ] Tab completion
- [ ] Hex viewer

## Building

### Client (Go)

```bash
cd client
go build -o edb .
```

### Agent (C)

```bash
cd agent
make
```

For cross-compilation (requires appropriate toolchain):
```bash
make CROSS=arm-linux-gnueabihf-
make CROSS=mipsel-linux-gnu-
```

## Usage

### Listen Mode (Recommended)

```bash
# On your workstation - start listening
./edb listen -p 1337

# On the device - connect back
./edb-agent -c 192.168.1.100:1337
```

### Connect Mode

```bash
# On the device - start listening
./edb-agent -l 1337

# On your workstation - connect to it
./edb shell 192.168.1.50:1337
```

## Getting the Agent onto a Device

Common bootstrap scenarios:

**Via UART/Serial Console:**
```bash
cd /tmp
wget http://192.168.1.100:8080/edb-agent-arm
chmod +x edb-agent-arm
./edb-agent-arm -c 192.168.1.100:1337
or
./edb-agent-arm -l 1337
```

**Via existing SSH/Telnet:**
```bash
tftp -g -r edb-agent-mips 192.168.1.100 8080
chmod +x edb-agent-mips
./edb-agent-arm -c 192.168.1.100:1337
or
./edb-agent-arm -l 1337
```

## License

MIT

---

*Necromancer Labs*
