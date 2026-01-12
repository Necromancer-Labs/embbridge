#!/bin/bash
#
# Build cross-compilation toolchains for embbridge agent
#
# Usage:
#   ./scripts/build-toolchains.sh          Build all toolchains
#   ./scripts/build-toolchains.sh arm      Build only ARM toolchain
#   ./scripts/build-toolchains.sh arm64    Build only ARM64 toolchain
#   ./scripts/build-toolchains.sh mips     Build only MIPS (big-endian) toolchain
#   ./scripts/build-toolchains.sh mipsel   Build only MIPS (little-endian) toolchain
#
# This uses buildroot to create musl-based static toolchains.
# First run takes 20-40 minutes per architecture.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILDROOT_DIR="$PROJECT_DIR/buildroot-2025.11"

# All supported architectures
ALL_ARCHS="arm arm64 mips mipsel"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

build_toolchain() {
    local arch=$1
    local config="edb_${arch}_defconfig"
    local output_dir="$BUILDROOT_DIR/output_${arch}"

    log_info "Building toolchain for: $arch"
    log_info "Output directory: $output_dir"

    if [ ! -f "$BUILDROOT_DIR/configs/$config" ]; then
        log_error "Config not found: $BUILDROOT_DIR/configs/$config"
        return 1
    fi

    cd "$BUILDROOT_DIR"

    # Configure
    log_info "Configuring $arch..."
    make O="$output_dir" "${config}"

    # Build just the toolchain (not full rootfs)
    log_info "Building $arch toolchain (this may take 20-40 minutes)..."
    make O="$output_dir" toolchain

    # Verify
    local gcc_path
    case $arch in
        arm)    gcc_path="$output_dir/host/bin/arm-buildroot-linux-musleabi-gcc" ;;
        arm64)  gcc_path="$output_dir/host/bin/aarch64-buildroot-linux-musl-gcc" ;;
        mips)   gcc_path="$output_dir/host/bin/mips-buildroot-linux-musl-gcc" ;;
        mipsel) gcc_path="$output_dir/host/bin/mipsel-buildroot-linux-musl-gcc" ;;
    esac

    if [ -x "$gcc_path" ]; then
        log_info "Successfully built $arch toolchain!"
        log_info "GCC: $gcc_path"
        $gcc_path --version | head -1
    else
        log_error "Toolchain build failed for $arch"
        return 1
    fi

    echo ""
}

show_usage() {
    echo "Usage: $0 [arch]"
    echo ""
    echo "Build cross-compilation toolchains for embbridge agent."
    echo ""
    echo "Arguments:"
    echo "  (none)    Build all toolchains (arm, arm64, mips, mipsel)"
    echo "  arm       Build ARM 32-bit toolchain"
    echo "  arm64     Build ARM 64-bit (AArch64) toolchain"
    echo "  mips      Build MIPS 32-bit big-endian toolchain"
    echo "  mipsel    Build MIPS 32-bit little-endian toolchain"
    echo ""
    echo "First build takes 20-40 minutes per architecture."
}

# Check buildroot exists
if [ ! -d "$BUILDROOT_DIR" ]; then
    log_error "Buildroot not found at: $BUILDROOT_DIR"
    exit 1
fi

# Parse arguments
if [ $# -eq 0 ]; then
    # Build all
    log_info "Building all toolchains: $ALL_ARCHS"
    echo ""
    for arch in $ALL_ARCHS; do
        build_toolchain "$arch"
    done
    log_info "All toolchains built successfully!"
elif [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    show_usage
    exit 0
else
    # Build specific arch
    arch=$1
    if [[ ! " $ALL_ARCHS " =~ " $arch " ]]; then
        log_error "Unknown architecture: $arch"
        log_error "Valid options: $ALL_ARCHS"
        exit 1
    fi
    build_toolchain "$arch"
    log_info "Done!"
fi
