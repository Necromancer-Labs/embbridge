# Command Reference

Complete list of embbridge shell commands organized by category.

## Navigation Commands

### ls

List directory contents.

**Usage:** `ls [path]`

**Arguments:**
- `path` - Directory to list (default: current directory)

**Example:**
```
edb[/]# ls /etc
drwxr-xr-x    4096  config/
-rw-r--r--    1234  passwd
-rw-r--r--     567  shadow
```

### cd

Change current working directory.

**Usage:** `cd <path>`

**Arguments:**
- `path` - Target directory (required)

**Example:**
```
edb[/]# cd /tmp
edb[/tmp]#
```

### pwd

Print current working directory.

**Usage:** `pwd`

**Example:**
```
edb[/tmp]# pwd
/tmp
```

### cat

Display file contents.

**Usage:** `cat <file>`

**Arguments:**
- `file` - Absolute path to file (required)

**Example:**
```
edb[/]# cat /etc/passwd
root:x:0:0:root:/root:/bin/sh
```

### realpath

Resolve path to canonical absolute form.

**Usage:** `realpath <path>`

**Arguments:**
- `path` - Path to resolve (required)

**Example:**
```
edb[/tmp]# realpath ../etc/passwd
/etc/passwd
```

## File Transfer Commands

### pull

Download a file from the device to your local machine.

**Usage:** `pull <remote-file> [local-path]`

**Arguments:**
- `remote-file` - Absolute path on device (required)
- `local-path` - Local destination (default: filename from remote)

**Example:**
```
edb[/]# pull /etc/passwd ./passwd.txt
Downloaded 1234 bytes to ./passwd.txt
```

### push

Upload a file from your local machine to the device.

**Usage:** `push <local-file> <remote-path>`

**Arguments:**
- `local-file` - Local file path (required)
- `remote-path` - Absolute destination path on device (required)

**Example:**
```
edb[/]# push ./script.sh /tmp/script.sh
Uploaded 567 bytes to /tmp/script.sh
```

## File Operation Commands

### rm

Remove a file or empty directory.

**Usage:** `rm <path>`

**Arguments:**
- `path` - Absolute path to remove (required)

**Example:**
```
edb[/]# rm /tmp/test.txt
```

### mv

Move or rename a file or directory.

**Usage:** `mv <src> <dst>`

**Arguments:**
- `src` - Absolute source path (required)
- `dst` - Absolute destination path (required)

**Example:**
```
edb[/]# mv /tmp/old.txt /tmp/new.txt
```

### cp

Copy a file.

**Usage:** `cp <src> <dst>`

**Arguments:**
- `src` - Absolute source path (required)
- `dst` - Absolute destination path (required)

**Example:**
```
edb[/]# cp /etc/passwd /tmp/passwd.bak
```

### mkdir

Create a directory.

**Usage:** `mkdir <path>`

**Arguments:**
- `path` - Absolute path for new directory (required)

**Example:**
```
edb[/]# mkdir /tmp/newdir
```

### chmod

Change file permissions.

**Usage:** `chmod <mode> <path>`

**Arguments:**
- `mode` - Octal permission mode (required)
- `path` - Absolute path to file (required)

**Example:**
```
edb[/]# chmod 755 /tmp/script.sh
```

## System Information Commands

### uname

Display system information (kernel, hostname, architecture).

**Usage:** `uname`

**Example:**
```
edb[/]# uname
Linux router 4.14.0 mips
```

### whoami

Display current user identity.

**Usage:** `whoami`

**Example:**
```
edb[/]# whoami
root (uid=0, gid=0)
```

### ps

Display process tree with PID, PPID, state, and command.

**Usage:** `ps`

**Example:**
```
edb[/]# ps
PID     PPID    STATE COMMAND
--------------------------------------------------------------------------------
1       0       S     /sbin/init
├── 234     1       S     /usr/sbin/dropbear
│   └── 567   234     S     -sh
```

### ss

Display network connections with process information.

**Usage:** `ss`

**Example:**
```
edb[/]# ss
PROTO  LOCAL ADDRESS          REMOTE ADDRESS         STATE        PID  PROCESS
------------------------------------------------------------------------------------------
tcp    0.0.0.0:22             0.0.0.0:*              LISTEN       234  dropbear
tcp    192.168.1.1:22         192.168.1.100:54321    ESTABLISHED  567  dropbear
```

### ip

Display network interfaces with MAC addresses, IP addresses, MTU, and state.

**Usage:** `ip`

**Example:**
```
edb[/]# ip
eth0: <UP,BROADCAST,RUNNING,MULTICAST> mtu 1500 state up
    link/ether aa:bb:cc:dd:ee:ff
    inet 192.168.1.1/24
lo: <UP,LOOPBACK,RUNNING> mtu 65536 state unknown
    inet 127.0.0.1/8
```

### ip-route

Display routing table.

**Usage:** `ip-route`

**Example:**
```
edb[/]# ip-route
default via 192.168.1.254 dev eth0 metric 100
192.168.1.0/24 dev eth0
```

### dmesg

Display kernel log messages.

**Usage:** `dmesg`

**Example:**
```
edb[/]# dmesg
[    0.000000] Linux version 4.14.0 ...
[    1.234567] eth0: link up
```

### cpuinfo

Display CPU information from /proc/cpuinfo.

**Usage:** `cpuinfo`

**Example:**
```
edb[/]# cpuinfo
system type     : MediaTek MT7621
processor       : 0
cpu model       : MIPS 1004Kc V2.15
```

### mtd

List MTD (flash) partitions.

**Usage:** `mtd`

**Example:**
```
edb[/]# mtd
dev:    size   erasesize  name
mtd0: 00030000 00010000 "u-boot"
mtd1: 00010000 00010000 "u-boot-env"
mtd2: 00f80000 00010000 "firmware"

Tip: Use 'pull /dev/mtdX' to download a partition
```

## Execution Commands

### exec

Execute a binary directly without shell interpretation.

**Usage:** `exec <command> [args...]`

**Arguments:**
- `command` - Binary path or command (required)
- `args` - Arguments to pass (optional)

**Example:**
```
edb[/]# exec /bin/ls -la /tmp
total 8
drwxrwxrwt    2 root     root          4096 Jan 11 12:00 .
drwxr-xr-x   18 root     root          4096 Jan 11 12:00 ..
```

### strings

Extract printable strings from a binary file.

**Usage:** `strings <file>`

**Arguments:**
- `file` - Absolute path to file (required)

**Example:**
```
edb[/]# strings /bin/busybox
BusyBox v1.30.1
Usage: busybox [function]
```

## System Control Commands

### reboot

Reboot the device.

**Usage:** `reboot`

**Example:**
```
edb[/]# reboot
Device is rebooting...
```

### kill-agent

Kill the agent's parent process (bind mode only). Useful for cleaning up after analysis.

**Usage:** `kill-agent`

**Example:**
```
edb[/]# kill-agent
Agent parent process (pid 1234) killed
```

## Shell Commands

### help

Display available commands.

**Usage:** `help`

### exit

Exit the shell and disconnect.

**Usage:** `exit`

### quit

Alias for exit.

**Usage:** `quit`
