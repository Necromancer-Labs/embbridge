# Architecture Reference

Resources for identifying embedded device architectures and selecting the correct edb-agent binary.

## Identifying Target Architecture

Before deploying edb-agent, determine the target device's CPU architecture:

```bash
# On the target device
cat /proc/cpuinfo
```

Look for keywords like:
- `ARMv7`, `ARMv5`, `Cortex-A` → use `edb-agent-arm`
- `AArch64`, `ARMv8` → use `edb-agent-arm64`
- `MIPS`, `mips32` with big-endian → use `edb-agent-mips`
- `MIPS`, `mips32` with little-endian → use `edb-agent-mipsel`

## Architecture to Binary Mapping

| Architecture | Endianness | Binary | Common Identifiers |
|--------------|------------|--------|-------------------|
| ARM 32-bit | Little | `edb-agent-arm` | ARMv5, ARMv6, ARMv7, Cortex-A7/A9/A15 |
| ARM 64-bit | Little | `edb-agent-arm64` | AArch64, ARMv8, Cortex-A53/A72 |
| MIPS 32-bit | Big | `edb-agent-mips` | MIPS32, mips-linux |
| MIPS 32-bit | Little | `edb-agent-mipsel` | MIPS32, mipsel-linux |
| x86 64-bit | Little | `edb-agent-x86_64` | x86_64, amd64 |

## Device Categories by Architecture

### ARM 32-bit (edb-agent-arm)

Common devices using ARM 32-bit processors:

- Smart TVs (Samsung, LG, Sony non-Android models)
- Older Android TVs (32-bit builds)
- Classic Raspberry Pi (Pi 1, Pi Zero)
- Consumer IoT cameras (TP-Link, Wyze v2)
- Legacy set-top boxes
- Many NVRs and DVRs
- Home automation hubs

### ARM 64-bit (edb-agent-arm64)

Common devices using ARM 64-bit processors:

- Modern Smart TVs (Android TV, Google TV, newer Tizen/WebOS)
- Raspberry Pi 3/4/5
- Modern Android phones and tablets
- ARM servers (Ampere, AWS Graviton)
- Newer NVR and IoT edge devices
- Modern set-top boxes

### MIPS Big-Endian (edb-agent-mips)

Common devices using MIPS big-endian:

- Older Cisco/Juniper enterprise routers
- Some legacy IP phones
- DSP boards
- Early enterprise networking appliances

### MIPS Little-Endian (edb-agent-mipsel)

Common devices using MIPS little-endian:

- Consumer Wi-Fi routers (D-Link DIR-645, Linksys WRT54GL variants)
- Atheros/Qualcomm-Atheros based SOHO access points
- Older NAS devices
- Many early OpenWrt hardware targets
- TP-Link routers
- Netgear consumer routers

## External Resources

Reference materials for researching embedded device architectures:

### ARM-Based Devices

**Wikipedia: List of products using ARM processors**

Catalogues devices (phones, TVs, household devices) that use ARM CPUs. Useful for associating consumer devices with ARM SoC cores.

- [List of products using ARM processors](https://en.wikipedia.org/wiki/List_of_products_using_ARM_processors)

### Microcontrollers and Embedded Processors

**Wikipedia: List of common microcontrollers**

Shows how architectures map to popular embedded families and brands (ARM Cortex-M, ESP32, 8051, etc.). Focused on MCUs but provides architecture context.

- [List of common microcontrollers](https://en.wikipedia.org/wiki/List_of_common_microcontrollers)

### Embedded Architecture Overviews

**AllPCB: Types of Embedded Microprocessors**

Overview of major embedded CPU architectures including ARM, MIPS, x86, and DSPs. Good for understanding what architectures appear in different product categories.

- [Types of Embedded Microprocessors](https://www.allpcb.com/allelectrohub/types-of-embedded-microprocessors-an-overview)

**AllPCB: Embedded Microprocessor Architectures**

Similar overview emphasizing mainstream architectures (ARM, x86, MIPS) commonly found in embedded systems.

- [Embedded Microprocessor Architectures](https://www.allpcb.com/allelectrohub)

## Tips for Architecture Identification

### When /proc/cpuinfo is unavailable

If you cannot access `/proc/cpuinfo`, try these alternatives:

```bash
# Check ELF binary headers
file /bin/busybox

# Look for architecture hints in firmware
strings /bin/busybox | grep -i "arm\|mips\|x86"

# Check kernel version string
uname -m
```

### Common SoC Vendors by Architecture

| Vendor | Typical Architecture |
|--------|---------------------|
| Broadcom (BCM) | ARM, MIPS |
| Qualcomm/Atheros | ARM, MIPS |
| MediaTek (MT) | ARM, MIPS |
| Realtek (RTL) | MIPS, ARM |
| Allwinner | ARM |
| Rockchip | ARM |
| Amlogic | ARM |
| HiSilicon | ARM |
| Ingenic | MIPS |

### Firmware Analysis Hints

When analyzing firmware images before deployment:

```bash
# Extract and check binaries
binwalk -e firmware.bin
file squashfs-root/bin/*

# Look for architecture in build strings
strings squashfs-root/bin/busybox | head -20
```
