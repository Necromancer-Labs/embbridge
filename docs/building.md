# Building from Source

Instructions for building embbridge agent and client from source.

## Prerequisites

### Client (Go)

- Go 1.21 or later

### Agent (C)

- GCC (for native builds)
- Buildroot toolchains (for cross-compilation)

## Building the Client

The client is written in Go and runs on your workstation.

### Basic Build

```bash
cd client
go build -o edb .
```

### Release Build

For smaller binaries without debug info or build paths:

```bash
cd client
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o edb .
```

Build flags explained:
- `CGO_ENABLED=0` - Pure Go build, no C dependencies
- `-trimpath` - Remove local file paths from binary
- `-ldflags="-s -w"` - Strip debug info and symbol table

## Building the Agent

The agent is written in C and runs on the target device.

### Native Build (x86_64)

```bash
cd agent
make              # Debug build
make release      # Release build (optimized, stripped)
```

Output: `agent/edb-agent`

### Cross-Compilation

First, build the toolchains (one-time setup, approximately 30 minutes per architecture):

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

Output directory: `agent/build/`

## Toolchain Details

The build system uses [Buildroot](https://buildroot.org/) to create musl-based cross-compilers.

### Binary Characteristics

- **Static linking** - No runtime dependencies
- **Small size** - 100-200KB per binary
- **Wide compatibility** - ARMv5 binary runs on ARMv5, v6, v7, and v8 (32-bit mode)

### Architecture Reference

| Target | Toolchain Prefix | Output Binary |
|--------|------------------|---------------|
| ARM 32-bit | arm-buildroot-linux-musleabi- | edb-agent-arm |
| ARM 64-bit | aarch64-buildroot-linux-musl- | edb-agent-arm64 |
| MIPS BE | mips-buildroot-linux-musl- | edb-agent-mips |
| MIPS LE | mipsel-buildroot-linux-musl- | edb-agent-mipsel |

### Toolchain Paths

After running `build-toolchains.sh`, toolchains are located at:

```
buildroot-2025.11/output_arm/host/bin/
buildroot-2025.11/output_arm64/host/bin/
buildroot-2025.11/output_mips/host/bin/
buildroot-2025.11/output_mipsel/host/bin/
```

## Verifying Binaries

### Check Architecture

```bash
file agent/build/edb-agent-arm
# ELF 32-bit LSB executable, ARM, EABI5 version 1 (SYSV), statically linked, stripped
```

### Verify Static Linking

```bash
ldd agent/build/edb-agent-arm
# not a dynamic executable
```

### Check for Embedded Paths

Ensure no local paths are embedded in binaries:

```bash
strings agent/build/edb-agent-arm | grep -i home
# Should return nothing

strings client/edb | grep -i home
# Should return nothing (if built with -trimpath)
```

## Troubleshooting

### Toolchain Not Found

If make fails with "toolchain not found":

```bash
# Check toolchain status
cd agent
make toolchain-status

# Rebuild specific toolchain
../scripts/build-toolchains.sh arm
```

### Go Module Issues

If Go build fails with module errors:

```bash
cd client
go mod tidy
go build -o edb .
```
