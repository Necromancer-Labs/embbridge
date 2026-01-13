# embbridge agent - build system
#
# Usage:
#   make              Build debug version (native)
#   make release      Build release version (native)
#   make arm          Build for ARM 32-bit
#   make arm64        Build for ARM 64-bit (AArch64)
#   make mips         Build for MIPS 32-bit big-endian
#   make mipsel       Build for MIPS 32-bit little-endian
#   make all-arch     Build all cross-compiled versions
#   make clean        Clean build artifacts
#
# Prerequisites for cross-compilation:
#   Run ../scripts/build-toolchains.sh first to build the toolchains

# =============================================================================
# Paths
# =============================================================================

PROJECT_ROOT := $(shell dirname $(CURDIR))
BUILDROOT := $(PROJECT_ROOT)/buildroot-2025.11

# Toolchain paths (from buildroot)
TC_ARM    := $(BUILDROOT)/output_arm/host/bin/arm-buildroot-linux-musleabi-
TC_ARM64  := $(BUILDROOT)/output_arm64/host/bin/aarch64-buildroot-linux-musl-
TC_MIPS   := $(BUILDROOT)/output_mips/host/bin/mips-buildroot-linux-musl-
TC_MIPSEL := $(BUILDROOT)/output_mipsel/host/bin/mipsel-buildroot-linux-musl-

# =============================================================================
# Compiler settings
# =============================================================================

CC = gcc
CFLAGS = -Wall -Wextra -std=c99 -I./include
LDFLAGS =

# Source files
SRCS = src/main.c \
       src/transport.c \
       src/protocol.c \
       src/commands/cmd_dispatch.c \
       src/commands/helpers.c \
       src/commands/basic_commands.c \
       src/commands/file_transfer.c \
       src/commands/file_operations.c \
       src/commands/system/uname.c \
       src/commands/system/ps.c \
       src/commands/system/exec.c \
       src/commands/system/ss.c \
       src/commands/system/kill_agent.c \
       src/commands/system/reboot.c \
       src/commands/system/whoami.c \
       src/commands/system/dmesg.c \
       src/commands/system/strings.c \
       src/commands/system/cpuinfo.c \
       src/commands/system/mtd.c \
       src/commands/system/ip.c

# Output directory for cross-compiled binaries
BUILD_DIR = build

# =============================================================================
# Native targets
# =============================================================================

TARGET = edb-agent

.PHONY: all release clean arm arm64 mips mipsel all-arch

# Debug build (native)
all: $(TARGET)

# Release build (native)
release: CFLAGS = -Wall -Wextra -std=c99 -I./include -Os -DNDEBUG
release: LDFLAGS = -s
release: $(TARGET)

$(TARGET): $(SRCS)
	$(CC) $(CFLAGS) -g -DDEBUG $(LDFLAGS) -o $@ $(SRCS)

# =============================================================================
# Cross-compilation targets
# =============================================================================

# Common flags for cross-compiled release builds
CROSS_CFLAGS = -Wall -Wextra -std=c99 -I./include -Os -DNDEBUG
CROSS_LDFLAGS = -static -s

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# ARM 32-bit
arm: $(BUILD_DIR)
	@if [ ! -x "$(TC_ARM)gcc" ]; then \
		echo "ERROR: ARM toolchain not found. Run: ../scripts/build-toolchains.sh arm"; \
		exit 1; \
	fi
	$(TC_ARM)gcc $(CROSS_CFLAGS) $(CROSS_LDFLAGS) -o $(BUILD_DIR)/edb-agent-arm $(SRCS)
	@echo "Built: $(BUILD_DIR)/edb-agent-arm"
	@ls -lh $(BUILD_DIR)/edb-agent-arm
	@file $(BUILD_DIR)/edb-agent-arm

# ARM 64-bit (AArch64)
arm64: $(BUILD_DIR)
	@if [ ! -x "$(TC_ARM64)gcc" ]; then \
		echo "ERROR: ARM64 toolchain not found. Run: ../scripts/build-toolchains.sh arm64"; \
		exit 1; \
	fi
	$(TC_ARM64)gcc $(CROSS_CFLAGS) $(CROSS_LDFLAGS) -o $(BUILD_DIR)/edb-agent-arm64 $(SRCS)
	@echo "Built: $(BUILD_DIR)/edb-agent-arm64"
	@ls -lh $(BUILD_DIR)/edb-agent-arm64
	@file $(BUILD_DIR)/edb-agent-arm64

# MIPS 32-bit big-endian
mips: $(BUILD_DIR)
	@if [ ! -x "$(TC_MIPS)gcc" ]; then \
		echo "ERROR: MIPS toolchain not found. Run: ../scripts/build-toolchains.sh mips"; \
		exit 1; \
	fi
	$(TC_MIPS)gcc $(CROSS_CFLAGS) $(CROSS_LDFLAGS) -o $(BUILD_DIR)/edb-agent-mips $(SRCS)
	@echo "Built: $(BUILD_DIR)/edb-agent-mips"
	@ls -lh $(BUILD_DIR)/edb-agent-mips
	@file $(BUILD_DIR)/edb-agent-mips

# MIPS 32-bit little-endian
mipsel: $(BUILD_DIR)
	@if [ ! -x "$(TC_MIPSEL)gcc" ]; then \
		echo "ERROR: MIPSEL toolchain not found. Run: ../scripts/build-toolchains.sh mipsel"; \
		exit 1; \
	fi
	$(TC_MIPSEL)gcc $(CROSS_CFLAGS) $(CROSS_LDFLAGS) -o $(BUILD_DIR)/edb-agent-mipsel $(SRCS)
	@echo "Built: $(BUILD_DIR)/edb-agent-mipsel"
	@ls -lh $(BUILD_DIR)/edb-agent-mipsel
	@file $(BUILD_DIR)/edb-agent-mipsel

# Build all architectures
all-arch: arm arm64 mips mipsel
	@echo ""
	@echo "All cross-compiled binaries:"
	@ls -lh $(BUILD_DIR)/

# =============================================================================
# Utility targets
# =============================================================================

clean:
	rm -f $(TARGET)
	rm -rf $(BUILD_DIR)

# Show toolchain status
toolchain-status:
	@echo "Toolchain status:"
	@echo -n "  ARM:    "; test -x "$(TC_ARM)gcc" && echo "OK ($(TC_ARM)gcc)" || echo "NOT FOUND"
	@echo -n "  ARM64:  "; test -x "$(TC_ARM64)gcc" && echo "OK ($(TC_ARM64)gcc)" || echo "NOT FOUND"
	@echo -n "  MIPS:   "; test -x "$(TC_MIPS)gcc" && echo "OK ($(TC_MIPS)gcc)" || echo "NOT FOUND"
	@echo -n "  MIPSEL: "; test -x "$(TC_MIPSEL)gcc" && echo "OK ($(TC_MIPSEL)gcc)" || echo "NOT FOUND"
	@echo ""
	@echo "To build toolchains, run: ../scripts/build-toolchains.sh"
