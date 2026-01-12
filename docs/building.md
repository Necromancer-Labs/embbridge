# Building from Source

## Prerequisites

- Go 1.21+ (for client)
- GCC (for native agent builds)
- Buildroot toolchains (for cross-compilation)

## Client (Go)

```bash
cd client
go build -o edb .
```

For a smaller binary:

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o edb .
```

## Agent (C)

### Native Build

```bash
cd agent
make              # Debug build
make release      # Release build (optimized, stripped)
```

### Cross-Compilation

First, build the toolchains (one-time, ~30 min per architecture):

```bash
./scripts/build-toolchains.sh
```

Then build for specific architectures:

```bash
cd agent
make arm          # ARM 32-bit (ARMv5+)
make arm64        # ARM 64-bit (AArch64)
make mips         # MIPS 32-bit big-endian
make mipsel       # MIPS 32-bit little-endian
make all-arch     # All architectures
```

Binaries are placed in `agent/build/`.

## Toolchain Details

The build system uses [Buildroot](https://buildroot.org/) to create musl-based cross-compilers. This produces:

- **Static binaries** — no runtime dependencies
- **Small size** — ~100-200KB per binary
- **Wide compatibility** — ARMv5 binary runs on ARMv5, v6, v7, and v8 (32-bit)

### Supported Architectures

| Target | Toolchain | Binary |
|--------|-----------|--------|
| ARM 32-bit | arm-buildroot-linux-musleabi- | edb-agent-arm |
| ARM 64-bit | aarch64-buildroot-linux-musl- | edb-agent-arm64 |
| MIPS BE | mips-buildroot-linux-musl- | edb-agent-mips |
| MIPS LE | mipsel-buildroot-linux-musl- | edb-agent-mipsel |

### Toolchain Location

After running `build-toolchains.sh`, toolchains are located at:

```
buildroot-2025.11/output_arm/host/bin/
buildroot-2025.11/output_arm64/host/bin/
buildroot-2025.11/output_mips/host/bin/
buildroot-2025.11/output_mipsel/host/bin/
```

## Verifying Binaries

Check architecture and linking:

```bash
file agent/build/edb-agent-arm
# ELF 32-bit LSB executable, ARM, EABI5 version 1 (SYSV), statically linked, stripped

ldd agent/build/edb-agent-arm
# not a dynamic executable (good - it's static)
```
