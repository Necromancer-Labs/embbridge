# embbridge (Embedded Debug Bridge)

embbridge — an ADB-like device bridge for embedded systems.
Shell, file transfer, and verified artifact bundles for forensics, investigation, and research.

---

## Problem

Embedded device access and investigation is fragmented chaos:
- Every vendor has different shell access (Telnet, SSH, proprietary)
- File transfer is often missing or inconsistent (scp, TFTP, wget, proprietary)
- No good way to collect firmware dumps and files

You UART into a router, get a busybox shell, and then... now what? How do you get files off? How do you pull the firmware? Every device is different and this is frustrating.

## Solution

One tool. One workflow. Every device.

```bash
# On your box
edb listen

# On device (after getting agent onto it)
./edb-agent -c 192.168.1.100:1337
```

Agent connects back, you're in. Interactive shell with:
```
edb> ls /
edb> cd /etc
edb> cat passwd
edb> get passwd /tmp/               # pull file to local
edb> put example.sh /tmp/example.sh # push file to device
edb> firmware                       # dump firmware partitions
edb> bundle --profile router        # collect standard artifacts
```

The main experience is **interactive exploration** — roaming around an embedded device like you would with a normal shell, but with excellent quality of life upgrades that make investigations much easier and standard. Saving you time and sanity. 

---

## Core Architecture

**Reverse mode** (agent connects out):
```
┌─────────────────────┐         ┌─────────────────────┐
│   edb client (Go)   │◄────────│   edb agent (C)     │
│   gocmd2 shell      │  msgpack│   tiny, static      │
│   your workstation  │         │   on the device     │
│                     │         │                     │
│   edb listen :1337  │◄────────│ ./edb-agent -c IP   │
└─────────────────────┘         └─────────────────────┘
```

**Bind mode** (you connect to agent):
```
┌─────────────────────┐         ┌─────────────────────┐
│   edb client (Go)   │────────►│   edb agent (C)     │
│   gocmd2 shell      │  msgpack│   tiny, static      │
│   your workstation  │         │   on the device     │
│                     │         │                     │
│   edb shell IP:1337 │────────►│ ./edb-agent -l 1337 │
└─────────────────────┘         └─────────────────────┘
```

**Reverse mode** works through NAT, firewalls, and weird network setups common in embedded research. **Bind mode** is simpler when you have direct network access to the device.

### Agent (C, runs on device)
- **Tiny**: Target <100KB static binary
- **No dependencies**: Static linking, runs on anything
- **Multi-arch**: ARM, MIPS, x86, RISC-V (cross-compiled)
- **Transports**: TCP (primary), serial (UART) later
- **No persistence**: Runs in /tmp, dies on reboot, that's fine
- **Simple**: Does what you tell it, nothing more

**Connection modes:**
```bash
# Reverse (agent connects to you) — recommended
./edb-agent -c 192.168.1.100:1337

# Bind (agent listens) — simpler setups
./edb-agent -l 1337
```

Reverse is default because:
- Works through NAT and firewalls
- Don't need to know device IP ahead of time
- More practical for most embedded scenarios

### Client (Go, runs on your box)
- **Interactive shell**: gocmd2 with readline for familiar shell experience
- **Config file**: `~/.edb/config.toml` for saved devices
- **Profiles directory**: `~/.edb/profiles/` for artifact collection
- **Output formats**: Interactive default, `--json` for scripting

**Client modes:**
```bash
# Listen for reverse connections (agent connects to you)
edb listen                    # listen on :1337
edb listen -p 4444            # listen on :4444

# Connect to agent in bind mode
edb shell 192.168.1.50:1337   # connect to device
```

### Protocol
- **MessagePack over TCP**: Compact, fast, C and Go libraries exist
- **Simple framing**: Length-prefixed messages
- **Optional encryption**: ChaCha20-Poly1305 with PSK (off by default)
- **No TLS**: Absurd for LAN/research use, TLS is also huge when dealing with embedded systems. 

---

## Interactive Shell Commands

When you run `edb shell <device>`, you get an interactive prompt:

```
edb> help

Navigation:
  ls [path]              List directory
  cd <path>              Change directory
  pwd                    Print working directory
  cat <file>             Print file contents
  hexdump <file>         Hex dump of file

File Transfer:
  get <remote> [local]   Download file/directory
  put <local> <remote>   Upload file

File Operations:
  rm <path>              Remove file
  mv <src> <dst>         Move/rename
  cp <src> <dst>         Copy
  mkdir <path>           Create directory
  chmod <mode> <path>    Change permissions
  touch <path>           Create empty file

Firmware:
  mtd                    List MTD partitions
  firmware               Dump all firmware partitions
  firmware <partition>   Dump specific partition

Artifacts:
  bundle                 Collect artifacts (default profile)
  bundle --profile <p>   Collect with specific profile

System:
  exec <cmd>             Run command on device (no shell)
  ps                     List processes (tree view)
  netstat                Network connections (with PIDs)
  env                    Environment variables
  uname                  System info

Session:
  kill-agent             Kill agent parent process (bind mode)
  exit                   Close connection
```

---

## Firmware Dumping

The killer feature for embedded research. One command to dump everything:

```
edb> mtd
NAME         SIZE       ERASESIZE
mtd0         256K       64K        bootloader
mtd1         64K        64K        config
mtd2         3M         64K        kernel
mtd3         12M        64K        rootfs

edb> firmware
Dumping mtd0 (bootloader)... done [256K]
Dumping mtd1 (config)... done [64K]
Dumping mtd2 (kernel)... done [3M]
Dumping mtd3 (rootfs)... done [12M]
Saved to ./firmware_devicename_2025-01-03/
```

Supports:
- MTD devices (`/dev/mtd*`)
- Block devices
- UBI volumes
- Raw flash reads where possible

---

## Artifact Bundles

Quick collection with profiles:

```
edb> bundle --profile router
```

Produces:
```
bundle_devicename_2025-01-03T14:32:00Z/
├── manifest.json
├── hashes.sha256
├── config/
│   ├── passwd
│   ├── shadow
│   ├── network/
│   └── ...
├── firmware/
│   ├── mtd0_bootloader.bin
│   ├── mtd1_config.bin
│   ├── mtd2_kernel.bin
│   └── mtd3_rootfs.bin
├── state/
│   ├── processes.txt
│   ├── netstat.txt
│   ├── mounts.txt
│   └── dmesg.txt
└── filesystem/
    └── (key directories)
```

### Profile System

Profiles are TOML files. Community can contribute profiles for specific device families:

```toml
# profiles/router.toml
name = "router"
description = "Generic router artifact collection"

[[files]]
path = "/etc/passwd"
output = "config/passwd"

[[files]]
path = "/etc/shadow"
output = "config/shadow"
optional = true

[[directories]]
path = "/etc/config"
output = "config/uci"
optional = true

[[commands]]
name = "processes"
command = "ps"
output = "state/processes.txt"

[[commands]]
name = "network"
command = "netstat -an"
output = "state/netstat.txt"

[[commands]]
name = "mounts"
command = "mount"
output = "state/mounts.txt"

[[firmware]]
collect = true  # dump all MTD/UBI partitions
```

Profiles become a community contribution point: "Here's the profile for Ubiquiti EdgeRouters." Network effects kick in.

---

## Getting the Agent onto a Device

The bootstrap problem. Common scenarios:

**Scenario 1: UART shell access (reverse connection)**
```bash
# On your box, start listening
edb listen

# On device via UART
cd /tmp
wget http://192.168.1.100:8080/edb-agent-mips
chmod +x edb-agent-mips
./edb-agent-mips -c 192.168.1.100:1337
```
Agent connects back, you're immediately in a shell.

**Scenario 2: Telnet/SSH access**
```bash
# On your box, serve the agent and listen
python3 -m http.server 8080 &
edb listen

# On device
tftp -g -r edb-agent-arm 192.168.1.100
chmod +x edb-agent-arm
./edb-agent-arm -c 192.168.1.100:1337
```

**Scenario 3: Bind mode (simpler, device listens)**
```bash
# On device
./edb-agent-arm -l 1337

# On your box
edb shell 192.168.1.50:1337
```

**Scenario 4: Already have file transfer**
```bash
# scp, ftp, whatever works
scp edb-agent-arm root@device:/tmp/
ssh root@device "/tmp/edb-agent-arm -c 192.168.1.100:1337"
```

---

## Non-Interactive Mode

For scripting, you can run one-off commands:

```bash
# Single command
edb exec router1 "cat /etc/passwd"

# Pull a file
edb get router1 /etc/passwd ./passwd

# Push a file
edb put router1 ./script.sh /tmp/script.sh

# Collect bundle
edb bundle router1 --profile router

# Dump firmware
edb firmware router1
```

---

## Project Structure

```
embbridge/
├── Makefile
├── README.md
├── LICENSE                    # MIT
│
├── agent/                     # C agent
│   ├── Makefile
│   ├── src/
│   │   ├── main.c
│   │   ├── protocol.c         # MessagePack handling
│   │   ├── commands.c         # Command implementations
│   │   ├── files.c            # File operations
│   │   ├── firmware.c         # MTD/UBI dumping
│   │   └── transport.c        # TCP/serial
│   └── include/
│       └── edb.h
│
├── client/                    # Go client
│   ├── go.mod
│   ├── main.go
│   ├── cmd/                   # CLI commands
│   ├── shell/                 # gocmd2 interactive shell
│   ├── protocol/              # MessagePack protocol
│   └── profiles/              # Profile loading
│
├── profiles/                  # Built-in artifact profiles
│   ├── router.toml
│   ├── camera.toml
│   ├── generic-linux.toml
│   └── openwrt.toml
│
├── scripts/
│   ├── cross-compile.sh       # Build for ARM, MIPS, etc.
│   └── release.sh
│
└── docs/
    ├── protocol.md
    └── profiles.md
```

---

## Implementation Phases

### Phase 0: Foundation
- [x] **Protocol spec**: Define MessagePack message format for requests/responses
- [x] **Agent skeleton (C)**: main.c with argument parsing (-c/-l), TCP connect/listen
- [x] **Client skeleton (Go)**: main.go with cobra CLI, `edb listen` and `edb shell` commands
- [x] **Basic handshake**: Agent connects, client accepts, exchange hello message
- [ ] **Cross-compilation**: Makefile with ARM, MIPS, x86_64 targets using musl
- [x] **One command working**: `ls` end-to-end (agent lists dir, sends response, client displays)

### Phase 1: Core
- [x] Interactive shell (`edb shell`) — using gocmd2 readline shell
- [x] Navigation: ls ✓, cd ✓, pwd ✓, cat ✓
- [x] File transfer: get ✓, put ✓
- [x] System info: uname ✓
- [x] File ops: rm ✓, mv ✓, cp ✓, mkdir ✓, chmod ✓
- [x] Process listing: ps ✓ (tree view)
- [x] Network connections: netstat ✓ (TCP/UDP with process info)
- [x] Command execution: exec ✓ (fork+execv, no shell dependency)
- [x] **Agent fork-on-accept**: In bind mode, fork child for each connection so agent persists
- [x] **kill-agent command**: Kill parent agent process from client (bind mode)
- [ ] Config file for saved devices

### Phase 2: Polish
- [ ] Firmware dumping (MTD, UBI, block devices)
- [ ] Artifact bundles
- [ ] Profile system
- [ ] Built-in profiles (router, camera, generic-linux)
- [ ] Hash manifest generation

### Phase 3: Super polish
- [ ] Serial transport (UART)
- [ ] Progress bars for file transfer
- [ ] Hex viewer command
- [ ] Tab completion for paths
- [x] Command history (readline)

### Phase 4: Ecosystem
- [ ] Profile contribution workflow
- [ ] More built-in profiles
- [ ] Documentation

---

## Technical Decisions

### Why C for the agent?
- Smallest possible binary (<100KB target)
- Runs on anything with a C compiler
- No runtime, no garbage collection
- Cross-compiles easily to obscure architectures
- Embedded developers already know C

### Why Go for the client?
- gocmd2 provides a modular command shell with readline support
- Easy to write, fast to iterate
- Good cross-platform support
- Single binary distribution
- MessagePack libraries available

### Why MessagePack?
- Compact binary format (smaller than JSON, no schema like protobuf)
- C library: msgpack-c (or hand-roll — it's simple enough)
- Go library: github.com/vmihailenco/msgpack
- Fast to parse, low overhead
- Self-describing enough to debug
- Simpler than protobuf (no .proto files, no codegen)

### Why no encryption by default?
- This is for LAN research/forensics
- You're already on the device via UART or local network
- TLS adds complexity and massive binary size
- Optional ChaCha20-Poly1305 with PSK if you want it later

### Licensing: MIT
- Maximum adoption, minimum friction
- Embedded companies can use it internally without legal review
- Simple, well-understood
- No patent clauses needed — this isn't novel enough to patent
- Community contributions stay simple

---

## Protocol Overview

Simple MessagePack messages with length prefix:

```
[4 bytes: length][msgpack payload]
```

Request format:
```json
{
  "cmd": "ls",
  "args": {
    "path": "/etc"
  }
}
```

Response format:
```json
{
  "ok": true,
  "data": {
    "entries": [...]
  }
}
```

Or error:
```json
{
  "ok": false,
  "error": "permission denied"
}
```

File transfers use chunked streaming with the same framing.

---

## Naming

**Project**: `embbridge` (Embedded Debug Bridge)
**Command**: `edb`

Why this works:
- **embbridge** is unique on GitHub — no collisions with edb-debugger, EnterpriseDB, etc.
- **edb** as the command echoes **adb** — instant recognition, muscle memory
- Short command (3 letters) for daily use, descriptive project name for discoverability
- "embedded debug bridge" parallels "android debug bridge" exactly

Usage pattern:
```bash
# Install
go install github.com/Necromancer-Labs/embbridge/client@latest

# Use the edb command
edb shell router1
edb bundle camera1 --profile ip-camera
```

---

## Competitive Positioning

| Feature | ADB | SSH | embbridge |
|---------|-----|-----|-----------|
| Interactive shell | Yes | Yes | **Yes (TUI)** |
| File transfer | Yes | SCP | **Yes (integrated)** |
| Firmware dump | No | No | **Yes** |
| Artifact bundles | No | No | **Yes** |
| Tiny agent | ~1MB | ~2MB+ | **<100KB** |
| Multi-arch | ARM only | Heavy | **Yes** |
| Profile system | No | No | **Yes** |
| Nice TUI | No | No | **Yes** |

---

## Success Metrics

**Phase 1 (MVP)**
- Works on 3 device families you already touch
- Agent compiles for ARM and MIPS under 100KB
- Interactive shell feels good

**Phase 2 (Useful)**
- Firmware dumping works on real devices
- Bundles used in actual research
- 3+ community profiles

**Phase 3 (Growth)**
- GitHub stars > 500
- External researchers using it
- "edb bundle" becomes a thing people say

---

## Next Steps

1. Set up C agent build with cross-compilation
2. Implement basic TCP transport + MessagePack
3. Build minimal Go client with bubbletea
4. Get `ls` and `cat` working end-to-end
5. Test on one real device

---

*"Every embedded device needs a bridge."*

*embbridge*
