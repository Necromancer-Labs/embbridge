# Protocol Specification

embbridge wire protocol documentation (Version 1).

## Overview

The embbridge protocol is a request-response protocol using MessagePack encoding over TCP. Designed for resource-constrained embedded devices while remaining easy to debug and extend.

## Wire Format

All messages use length-prefixed framing:

```
+----------------+------------------+
| Length (4B BE) | MessagePack Data |
+----------------+------------------+
```

- **Length**: 4 bytes, big-endian, unsigned. Size of MessagePack payload only.
- **MessagePack Data**: The message encoded as MessagePack.
- **Maximum message size**: 16 MB (16777216 bytes)

## Connection Modes

### Bind Mode

Agent listens, client connects.

```
1. Agent listens on port (default: 1337)
2. Client connects to agent
3. Client sends "hello"
4. Agent responds with "hello_ack"
5. Connection established
```

### Reverse Mode

Client listens, agent connects out.

```
1. Client listens on port (default: 1337)
2. Agent connects to client
3. Agent sends "hello"
4. Client responds with "hello_ack"
5. Connection established
```

## Message Types

### hello

Handshake initiation message.

```json
{
  "type": "hello",
  "version": 1,
  "agent": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always "hello" |
| version | int | Protocol version (currently 1) |
| agent | bool | true if sender is agent, false if client |

### hello_ack

Handshake response message.

```json
{
  "type": "hello_ack",
  "version": 1,
  "agent": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always "hello_ack" |
| version | int | Protocol version (currently 1) |
| agent | bool | true if sender is agent, false if client |

### req

Command request from client to agent.

```json
{
  "type": "req",
  "id": 1,
  "cmd": "ls",
  "args": {"path": "/etc"}
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always "req" |
| id | uint32 | Request ID for matching responses |
| cmd | string | Command name |
| args | map | Command-specific arguments |

### resp

Command response from agent to client.

```json
{
  "type": "resp",
  "id": 1,
  "ok": true,
  "data": {"entries": [...]},
  "error": ""
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always "resp" |
| id | uint32 | Matches request ID |
| ok | bool | true if command succeeded |
| data | map | Response data (when ok=true) |
| error | string | Error message (when ok=false) |

### data

Chunked data transfer message.

```json
{
  "type": "data",
  "id": 1,
  "seq": 0,
  "data": "<binary>",
  "done": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Always "data" |
| id | uint32 | Matches request ID |
| seq | uint32 | Sequence number (0-indexed) |
| data | binary | Data chunk (max 64KB) |
| done | bool | true if last chunk |

## Commands

### Navigation Commands

#### ls

List directory contents.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Directory path |

**Response data:**
```json
{
  "entries": [
    {"name": "passwd", "type": "file", "size": 1234, "mode": 420, "mtime": 1704307200}
  ]
}
```

Entry types: `file`, `dir`, `link`, `other`

#### pwd

Get current working directory.

**Request args:** none

**Response data:**
```json
{"path": "/tmp"}
```

#### cd

Change current directory.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Target directory |

**Response data:**
```json
{"path": "/etc"}
```

#### cat

Read file contents.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | File path |

**Response data:**
```json
{"content": "<binary>"}
```

#### realpath

Resolve path to canonical form.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Path to resolve |

**Response data:**
```json
{"path": "/etc/passwd"}
```

### File Transfer Commands

#### pull

Download file from device.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Remote file path |

**Response:** Initial response with file info, followed by data messages.

```json
{"size": 1234, "mode": 420}
```

#### push

Upload file to device.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Remote destination path |
| size | uint64 | yes | File size in bytes |
| mode | uint32 | yes | File permissions |

**Response:** Acknowledgment, then client sends data messages.

### File Operation Commands

#### rm

Remove file or empty directory.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Path to remove |

#### mv

Move or rename file.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| src | string | yes | Source path |
| dst | string | yes | Destination path |

#### cp

Copy file.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| src | string | yes | Source path |
| dst | string | yes | Destination path |

#### mkdir

Create directory.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | Directory path |
| mode | uint32 | yes | Directory permissions |

#### chmod

Change file permissions.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | File path |
| mode | uint32 | yes | New permissions |

### System Information Commands

#### uname

Get system information.

**Request args:** none

**Response data:**
```json
{
  "sysname": "Linux",
  "nodename": "router",
  "release": "4.14.0",
  "machine": "mips"
}
```

#### whoami

Get current user identity.

**Request args:** none

**Response data:**
```json
{"user": "root", "uid": 0, "gid": 0}
```

#### ps

Get process list.

**Request args:** none

**Response data:**
```json
{
  "processes": [
    {"pid": 1, "ppid": 0, "name": "init", "state": "S", "cmdline": "/sbin/init"}
  ]
}
```

#### ss

Get network connections.

**Request args:** none

**Response data:**
```json
{
  "connections": [
    {"proto": "tcp", "local_addr": "0.0.0.0", "local_port": 22, "remote_addr": "0.0.0.0", "remote_port": 0, "state": "LISTEN", "pid": 234, "process": "dropbear"}
  ]
}
```

#### ip_addr

Get network interface information.

**Request args:** none

**Response data:**
```json
{"content": "<binary>"}
```

#### ip_route

Get routing table.

**Request args:** none

**Response data:**
```json
{"content": "<binary>"}
```

#### dmesg

Get kernel log.

**Request args:** none

**Response data:**
```json
{"log": "<binary>"}
```

#### cpuinfo

Get CPU information.

**Request args:** none

**Response data:**
```json
{"content": "<binary>"}
```

#### mtd

Get MTD partition list.

**Request args:** none

**Response data:**
```json
{"content": "<binary>"}
```

#### strings

Extract printable strings from file.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| path | string | yes | File path |
| min_len | int | no | Minimum string length (default: 4) |

**Response data:**
```json
{"content": "<binary>"}
```

### Execution Commands

#### exec

Execute binary directly.

**Request args:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | yes | Command with arguments |

**Response data:**
```json
{"stdout": "<binary>", "stderr": "<binary>", "exitcode": 0}
```

### System Control Commands

#### reboot

Reboot the device.

**Request args:** none

**Response:** Acknowledgment before reboot.

#### kill-agent

Kill agent's parent process.

**Request args:** none

**Response data:**
```json
{"killed_pid": 1234}
```

## Error Responses

Common error strings in the `error` field:

| Error | Description |
|-------|-------------|
| not found | File or directory does not exist |
| permission denied | Insufficient permissions |
| is a directory | Expected file, got directory |
| not a directory | Expected directory, got file |
| invalid path | Malformed path |
| io error | Generic I/O error |
| out of memory | Memory allocation failed |
| unknown command | Command not recognized |
