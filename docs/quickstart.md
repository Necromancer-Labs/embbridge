# Quick Start

Get embbridge running in 5 minutes.

## 1. Get the Agent

**Option A: Download pre-built binary**

```bash
# From GitHub releases
wget https://github.com/Necromancer-Labs/embbridge/releases/latest/download/edb-agent-arm
```

**Option B: Build from source**

```bash
cd agent && make arm    # or: make mipsel, make arm64, etc.
```

See [Building](building.md) for full instructions.

## 2. Transfer Agent to Device

Use whatever method the device supports:

```bash
# via wget (serve from workstation, fetch from device)
Workstation: python -m http.server 8080
Device:      wget -O /tmp/edb http://192.168.1.100:8080/edb-agent-arm

# via scp
scp edb-agent-arm root@192.168.1.50:/tmp/

# via netcat (listen on device, push from workstation)
Device:      nc -l -p 4444 > /tmp/edb
Workstation: nc 192.168.1.50 4444 < edb-agent-arm

# via tftp
Device: tftp -g -r edb-agent-arm 192.168.1.100
Workstation: sudo ./utftp 

You can use our `ultra tftp` binary found here: https://github.com/Necromancer-Labs/utftp
```

## 3. Start the Agent

```bash
# On the device
chmod +x /tmp/edb
/tmp/edb -l 1337
```

## 4. Connect from Workstation

```bash
./edb shell 192.168.1.50:1337
```

That's it. You now have a consistent interface with file transfer, process listing, and more.

## Connection Modes

**Bind mode** — agent listens, you connect to it:

```bash
# Device
./edb-agent -l 1337

# Workstation
./edb shell 192.168.1.50:1337
```

**Reverse mode** — agent connects out to you (useful for NAT/firewalls):

```bash
# Workstation
./edb listen

# Device
./edb-agent -c 192.168.1.100:1337
```

## Next Steps

- [Commands](commands.md) — Full command reference
- [Building](building.md) — Build for other architectures
