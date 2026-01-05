# embbridge Protocol Specification

Version: 0.1.0

## Overview

The embbridge protocol is a simple request-response protocol using MessagePack encoding over TCP. It's designed to be lightweight enough for resource-constrained embedded devices while remaining easy to debug and extend.

## Wire Format

All messages use a very simple length-prefixed framing:

```
+----------------+------------------+
| Length (4B BE) | MessagePack Data |
+----------------+------------------+
```

- **Length**: 4 bytes, big-endian, unsigned. Size of the MessagePack payload only (not including the length field itself). 
- **MessagePack Data**: The actual message, encoded as MessagePack.

Maximum message size: 16 MB (0x01000000 bytes). Messages larger than this should use chunked transfer.

## Connection Establishment

### Reverse Mode (Agent connects to Client)

1. Client listens on port (default: 1337)
2. Agent connects to client
3. Agent sends `hello` message
4. Client responds with `hello_ack`
5. Connection established, client can send commands

### Bind Mode (Client connects to Agent)

1. Agent listens on port (default: 1337)
2. Client connects to agent
3. Client sends `hello` message
4. Agent responds with `hello_ack`
5. Connection established, client can send commands

## Message Types

### Hello (handshake)

Sent by the connecting party to initiate the session.

```
{
  "type": "hello",
  "version": 1,
  "agent": true|false
}
```

- `version`: Protocol version (currently 1)
- `agent`: true if this is the agent, false if client

### Hello Ack (handshake response)

```
{
  "type": "hello_ack",
  "version": 1,
  "agent": true|false
}
```

### Request (client → agent)

All commands from client to agent use this format:

```
{
  "type": "req",
  "id": <uint32>,
  "cmd": "<command_name>",
  "args": { ... }
}
```

- `id`: Request ID, used to match responses. Monotonically increasing.
- `cmd`: Command name ("ls", "cat", "get", "put")
- `args`: Command-specific arguments (can be an empty map)

### Response (agent → client)

```
{
  "type": "resp",
  "id": <uint32>,
  "ok": true|false,
  "data": { ... },
  "error": "<error message>"
}
```

- `id`: Matches the request ID
- `ok`: true if command succeeded, false otherwise
- `data`: Command-specific response data (present if ok=true)
- `error`: Error message (present if ok=false)

### Data (chunked transfer)

For large file transfers, data is sent in chunks:

```
{
  "type": "data",
  "id": <uint32>,
  "seq": <uint32>,
  "data": <binary>,
  "done": true|false
}
```

- `id`: Matches the request ID
- `seq`: Sequence number (0-indexed)
- `data`: Binary chunk (max 64KB per chunk)
- `done`: true if this is the last chunk

## Commands

### ls - List Directory

**Request:**
```
{
  "cmd": "ls",
  "args": {
    "path": "/etc"
  }
}
```

**Response:**
```
{
  "ok": true,
  "data": {
    "entries": [
      {
        "name": "passwd",
        "type": "file",
        "size": 1234,
        "mode": 420,
        "mtime": 1704307200
      },
      {
        "name": "config",
        "type": "dir",
        "size": 4096,
        "mode": 493,
        "mtime": 1704307200
      }
    ]
  }
}
```

Entry types: `"file"`, `"dir"`, `"link"`, `"other"`

### cat - Read File

**Request:**
```
{
  "cmd": "cat",
  "args": {
    "path": "/etc/passwd"
  }
}
```

**Response (small files, <64KB):**
```
{
  "ok": true,
  "data": {
    "content": <binary>
  }
}
```

**Response (large files):**
Initial response indicates chunked transfer, followed by `data` messages.

### pwd - Print Working Directory

**Request:**
```
{
  "cmd": "pwd",
  "args": {}
}
```

**Response:**
```
{
  "ok": true,
  "data": {
    "path": "/tmp"
  }
}
```

### cd - Change Directory

**Request:**
```
{
  "cmd": "cd",
  "args": {
    "path": "/etc"
  }
}
```

**Response:**
```
{
  "ok": true,
  "data": {
    "path": "/etc"
  }
}
```

### get - Download File

**Request:**
```
{
  "cmd": "get",
  "args": {
    "path": "/etc/passwd"
  }
}
```

**Response:**
Chunked transfer using `data` messages.

### put - Upload File

**Request:**
```
{
  "cmd": "put",
  "args": {
    "path": "/tmp/file.txt",
    "size": 1234,
    "mode": 420
  }
}
```

**Response:**
```
{
  "ok": true,
  "data": {}
}
```

After response, client sends `data` messages with file content.

### exec - Execute Command

**Request:**
```
{
  "cmd": "exec",
  "args": {
    "command": "ps aux"
  }
}
```

**Response:**
```
{
  "ok": true,
  "data": {
    "stdout": <binary>,
    "stderr": <binary>,
    "exitcode": 0
  }
}
```

### mkdir - Create Directory

**Request:**
```
{
  "cmd": "mkdir",
  "args": {
    "path": "/tmp/newdir",
    "mode": 493
  }
}
```

### rm - Remove File/Directory

**Request:**
```
{
  "cmd": "rm",
  "args": {
    "path": "/tmp/file.txt",
    "recursive": false
  }
}
```

### mv - Move/Rename

**Request:**
```
{
  "cmd": "mv",
  "args": {
    "src": "/tmp/old.txt",
    "dst": "/tmp/new.txt"
  }
}
```

### chmod - Change Mode

**Request:**
```
{
  "cmd": "chmod",
  "args": {
    "path": "/tmp/file.txt",
    "mode": 493
  }
}
```

### uname - System Info

**Request:**
```
{
  "cmd": "uname",
  "args": {}
}
```

**Response:**
```
{
  "ok": true,
  "data": {
    "sysname": "Linux",
    "nodename": "router",
    "release": "4.14.0",
    "version": "#1 SMP",
    "machine": "mips"
  }
}
```

## Error Codes

Common error strings returned in the `error` field:

- `"not found"` - File/directory does not exist
- `"permission denied"` - Insufficient permissions
- `"is a directory"` - Expected file, got directory
- `"not a directory"` - Expected directory, got file
- `"invalid path"` - Malformed path
- `"io error"` - Generic I/O error
- `"unknown command"` - Command not recognized

## Future Extensions

- `mtd` - List MTD partitions
- `firmware` - Dump firmware
- `bundle` - Collect artifacts
- `ps` - Process list
- `netstat` - Network connections
