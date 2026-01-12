# Command Reference

All commands available in the embbridge shell.

## Navigation

### ls [path]

List directory contents.

```
edb> ls /etc
drwxr-xr-x    4096  config/
-rw-r--r--    1234  passwd
-rw-r--r--     567  shadow
```

### cd \<path\>

Change current directory.

```
edb> cd /tmp
/tmp
```

### pwd

Print current working directory.

```
edb> pwd
/tmp
```

### cat \<file\>

Display file contents.

```
edb> cat /etc/passwd
root:x:0:0:root:/root:/bin/sh
```

## File Transfer

### pull \<remote\> [local]

Download a file from the device.

```
edb> pull /etc/passwd ./passwd.txt
Downloaded 1234 bytes to ./passwd.txt
```

### push \<local\> \<remote\>

Upload a file to the device.

```
edb> push ./exploit.sh /tmp/exploit.sh
Uploaded 567 bytes to /tmp/exploit.sh
```

## File Operations

### rm \<path\>

Remove a file.

```
edb> rm /tmp/test.txt
```

### mv \<src\> \<dst\>

Move or rename a file.

```
edb> mv /tmp/old.txt /tmp/new.txt
```

### cp \<src\> \<dst\>

Copy a file.

```
edb> cp /etc/passwd /tmp/passwd.bak
```

### mkdir \<path\>

Create a directory.

```
edb> mkdir /tmp/newdir
```

### chmod \<mode\> \<path\>

Change file permissions.

```
edb> chmod 755 /tmp/script.sh
```

## System Information

### uname

Display system information.

```
edb> uname
Linux router 4.14.0 #1 SMP mips
```

### whoami

Display current user.

```
edb> whoami
root (uid=0, gid=0)
```

### ps

Display process tree.

```
edb> ps
  PID   PPID  USER      COMMAND
    1      0  root      /sbin/init
  234      1  root      /usr/sbin/dropbear
  567    234  root      -sh
```

### ss

Display network connections with PIDs.

```
edb> ss
Proto  Local Address       Remote Address      State       PID/Program
tcp    0.0.0.0:22          0.0.0.0:*           LISTEN      234/dropbear
tcp    192.168.1.1:22      192.168.1.100:54321 ESTABLISHED 567/dropbear
```

### dmesg

Display kernel log.

```
edb> dmesg
[    0.000000] Linux version 4.14.0 ...
[    1.234567] eth0: link up
```

### cpuinfo

Display CPU information.

```
edb> cpuinfo
system type     : MediaTek MT7621
processor       : 0
cpu model       : MIPS 1004Kc V2.15
```

### mtd

List MTD (flash) partitions.

```
edb> mtd
dev:    size   erasesize  name
mtd0: 00030000 00010000 "u-boot"
mtd1: 00010000 00010000 "u-boot-env"
mtd2: 00f80000 00010000 "firmware"

Tip: Use 'pull /dev/mtdX' to download a partition
```

## Execution

### exec \<command\> [args...]

Execute a binary directly (no shell).

```
edb> exec /bin/ls -la /tmp
total 8
drwxrwxrwt    2 root     root          4096 Jan 11 12:00 .
drwxr-xr-x   18 root     root          4096 Jan 11 12:00 ..
```

### strings \<file\>

Extract printable strings from a file.

```
edb> strings /bin/busybox
BusyBox v1.30.1
Usage: busybox [function]
```

## System Control

### reboot

Reboot the device.

```
edb> reboot
Rebooting device...
```

## Shell Control

### help

Display available commands.

### exit / quit

Exit the shell.
