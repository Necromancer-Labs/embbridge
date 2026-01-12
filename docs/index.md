# embbridge

**Embedded Debug Bridge** — adb, but for embedded systems.

A lightweight agent/client tool for interacting with embedded devices. Useful for firmware analysis, security research, and device forensics.

## Why embbridge?

Android has `adb`. Embedded Linux has nothing. Every device is different — different shells, different utilities, different transfer methods. embbridge gives you a consistent interface that works everywhere.

- **One agent binary** — statically linked, ~100KB, runs anywhere
- **Zero dependencies** — doesn't need busybox, shell, or anything on target
- **Consistent interface** — same commands work on every device

## Quick Links

- [Quick Start](quickstart.md) — Get up and running in 5 minutes
- [Commands](commands.md) — Full command reference
- [Building](building.md) — Build from source
- [Protocol](protocol.md) — Wire protocol specification

## Supported Architectures

| Binary | Compatibility | Typical Devices |
|--------|---------------|-----------------|
| `edb-agent-arm` | ARMv5+ (v5, v6, v7, v8 32-bit) | Raspberry Pi, routers, IoT |
| `edb-agent-arm64` | AArch64 | Modern ARM systems |
| `edb-agent-mips` | MIPS32 big-endian | Broadcom/Atheros routers |
| `edb-agent-mipsel` | MIPS32 little-endian | Consumer routers |

## License

MIT — [Necromancer Labs](https://github.com/Necromancer-Labs)
