# Graveyard

Graveyard is the TUI for managing embbridge agents. It uses a hybrid architecture that combines two different terminal UI approaches for optimal user experience.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     GRAVEYARD                               │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌─────────────┐         ┌─────────────────────────────┐   │
│   │  Bubble Tea │ ──────► │         gocmd2              │   │
│   │  Dashboard  │         │     Interactive Shell       │   │
│   └─────────────┘         └─────────────────────────────┘   │
│         ▲                           │                       │
│         │                           │                       │
│         └───────── exit ────────────┘                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Bubble Tea Dashboard

The device list view uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a visual TUI experience:

- Displays all connected and previously-seen devices
- Status badges: `● LIVE`, `○ DEAD`, `● BUSY`, `↔ XFER`
- Keyboard navigation with vim bindings (j/k)
- Custom device naming, colors, and architecture labels
- Real-time updates via event channel

### gocmd2 Shell

When you select a device, graveyard switches to [gocmd2](https://github.com/Necromancerlabs/gocmd2) for an interactive shell:

- Readline-based REPL with command history
- Tab completion for commands and remote file paths
- Filesystem commands: `ls`, `cd`, `pwd`, `cat`, `rm`, `mv`, `cp`, `mkdir`
- System commands: `ps`, `ss`, `uname`, `whoami`, `dmesg`
- File transfers: `pull`, `push` with progress tracking
- Type `exit` to return to the dashboard

This hybrid approach solves viewport/buffer issues by using readline's native terminal handling for the shell, while keeping the visual polish of Bubble Tea for the device list.

## Persistent Device Storage

Graveyard remembers your devices between sessions. Device data is saved to `~/.graveyard_devices.yaml`:

- Previously connected devices appear on startup (as `DEAD`)
- Custom names, colors, and architecture labels are preserved
- Press Enter on a dead device to attempt reconnection
- Press `d` on a dead device to remove it permanently

## Keyboard Shortcuts

### Shell

| Command | Description |
|---------|-------------|
| `ls [path]` | List directory contents |
| `cd <path>` | Change directory |
| `cat <file>` | Display file contents |
| `pull <remote> [local]` | Download file from device |
| `push <local> <remote>` | Upload file to device |
| `ps` | List processes |
| `exit` | Return to dashboard |

## Project Structure

```
tui/
├── main.go                      # Entry point, mode switching
├── internal/
│   ├── tui/                     # Bubble Tea device list
│   │   ├── model.go             # State and event handling
│   │   └── view.go              # Rendering
│   │
│   ├── shell/                   # gocmd2 interactive shell
│   │   ├── shell.go             # Shell setup and prompt
│   │   ├── completer.go         # Path tab completion
│   │   └── commands/            # Command implementations
│   │
│   ├── connection/              # Device management
│   │   ├── manager.go           # TCP listener, device tracking
│   │   ├── device.go            # Device state
│   │   ├── session.go           # Protocol wrapper
│   │   └── storage.go           # YAML persistence
│   │
│   └── ui/theme/
│       └── theme.go             # Lipgloss color theme
```

## Usage

```bash
# Start graveyard (listens on port 1337 by default, so do agents)
./graveyard

# Listen on a different port
./graveyard -port 9999
```