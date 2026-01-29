# Path Autocomplete in Graveyard

Graveyard provides intelligent tab completion for remote file paths. When you press TAB while typing a path argument, it queries the connected device and returns matching files/directories.

## Quick Demo

```
graveyard[router]:/# ls /e<TAB>
graveyard[router]:/# ls /etc/

graveyard[router]:/# cd /etc/net<TAB>
graveyard[router]:/# cd /etc/network/

graveyard[router]:/# cat /etc/pass<TAB>
graveyard[router]:/# cat /etc/passwd
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Input                               │
│                      ls /etc/pa<TAB>                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      readline Library                            │
│                                                                  │
│  Detects TAB keypress, calls AutoCompleter.Do(line, pos)        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PathCompleter.Do()                          │
│                                                                  │
│  1. Parse line: "ls /etc/pa"                                    │
│     └─ cmd = "ls", partial = "/etc/pa"                          │
│                                                                  │
│  2. Check if "ls" takes path arguments                          │
│     └─ pathCmds["ls"] = [0]  ✓ (first arg is a path)           │
│                                                                  │
│  3. Check if current arg position matches                       │
│     └─ We're on arg 0, and 0 is in [0]  ✓                      │
│                                                                  │
│  4. Split partial into dir + prefix                             │
│     └─ dir = "/etc", prefix = "pa"                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    completePath("/etc/pa")                       │
│                                                                  │
│  Query remote device: session.Ls("/etc")                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Remote Device                               │
│                                                                  │
│  Returns directory listing:                                      │
│  [passwd, passwd-, shadow, group, hostname, hosts, ...]         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Filter & Format                               │
│                                                                  │
│  Filter entries starting with "pa":                             │
│  └─ passwd, passwd-                                             │
│                                                                  │
│  Return completions (suffix only):                              │
│  └─ ["sswd", "sswd-"]                                           │
│                                                                  │
│  readline replaces "pa" with "passwd"                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Updated Input                               │
│                    ls /etc/passwd                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Supported Commands

The completer knows which commands take path arguments and at which positions:

| Command | Path Positions | Example |
|---------|---------------|---------|
| `ls`    | 0             | `ls <path>` |
| `cd`    | 0             | `cd <path>` |
| `cat`   | 0             | `cat <file>` |
| `rm`    | 0             | `rm <path>` |
| `mkdir` | 0             | `mkdir <path>` |
| `mv`    | 0, 1          | `mv <src> <dst>` |
| `cp`    | 0, 1          | `cp <src> <dst>` |
| `chmod` | 1             | `chmod 755 <path>` |
| `pull`  | 0             | `pull <remote> [local]` |
| `push`  | 1             | `push <local> <remote>` |
| `strings` | 0           | `strings <file>` |

### Position Mapping

- Position `0` = first argument after command
- Position `1` = second argument after command

For `chmod 755 /etc/foo`:
- `755` is position 0 (not a path)
- `/etc/foo` is position 1 (is a path)

---

## Implementation Details

### PathCompleter Struct

```go
type PathCompleter struct {
    shell        shellapi.ShellAPI      // Access to shell state
    cmdCompleter readline.AutoCompleter // Fallback for command names
    pathCmds     map[string][]int       // cmd -> path arg positions
}
```

### The Do() Method

```go
func (c *PathCompleter) Do(line []rune, pos int) ([][]rune, int) {
    lineStr := string(line[:pos])
    parts := strings.Fields(lineStr)

    // Still typing command name? Use default completer
    if len(parts) <= 1 && !strings.HasSuffix(lineStr, " ") {
        return c.cmdCompleter.Do(line, pos)
    }

    // Check if this command/position needs path completion
    cmd := parts[0]
    pathPositions, ok := c.pathCmds[cmd]
    if !ok {
        return nil, 0
    }

    // Determine current argument index
    argIndex := len(parts) - 1
    if strings.HasSuffix(lineStr, " ") {
        argIndex++
    }

    // Is this a path argument?
    argPos := argIndex - 1  // 0-based position
    for _, p := range pathPositions {
        if p == argPos {
            partial := ""
            if argIndex < len(parts) {
                partial = parts[argIndex]
            }
            return c.completePath(partial)
        }
    }

    return nil, 0
}
```

### Path Completion Logic

```go
func (c *PathCompleter) completePath(partial string) ([][]rune, int) {
    // Split "/etc/pa" into dir="/etc" and prefix="pa"
    dir := filepath.Dir(partial)
    prefix := filepath.Base(partial)

    // Query remote device
    session := c.getSession()
    resp, err := session.Ls(dir)
    if err != nil {
        return nil, 0
    }

    // Filter matching entries
    entries := resp.Data["entries"].([]interface{})
    var matches [][]rune

    for _, e := range entries {
        entry := e.(map[string]interface{})
        name := entry["name"].(string)
        entryType := entry["type"].(string)

        if strings.HasPrefix(name, prefix) {
            suffix := name[len(prefix):]
            if entryType == "dir" {
                suffix += "/"  // Add trailing slash for dirs
            }
            matches = append(matches, []rune(suffix))
        }
    }

    return matches, len(prefix)
}
```

---

## Edge Cases Handled

### Empty Input
```
ls <TAB>  →  Lists current directory contents
```

### Trailing Slash
```
ls /etc/<TAB>  →  Lists /etc/ contents (prefix = "")
```

### Relative Paths
```
ls foo<TAB>      →  Completes in current directory
ls ./bar<TAB>    →  Completes in current directory
ls ../baz<TAB>   →  Completes in parent directory
```

### Multiple Path Arguments
```
mv /etc/pa<TAB>       →  Completes first path
mv /etc/passwd /tmp/b<TAB>  →  Completes second path
```

### Non-Path Arguments
```
chmod 7<TAB>          →  No completion (position 0 is mode, not path)
chmod 755 /etc/p<TAB> →  Completes path (position 1)
```

---

## Integration with readline

The completer is hooked into readline during shell initialization:

```go
func setupPathCompletion(sh *shell.Shell) {
    rl := sh.GetReadline()

    // Get existing command completer
    cmdCompleter := rl.Config.AutoComplete

    // Wrap it with our path completer
    pathCompleter := NewPathCompleter(sh, cmdCompleter)

    // Replace the autocomplete handler
    rl.Config.AutoComplete = pathCompleter
}
```

This preserves command name completion (first word) while adding path completion for arguments.

---

## Performance Considerations

- **Network latency**: Each TAB press queries the remote device
- **Fail silently**: Network errors return empty completions (no hang)
- **No caching**: Fresh results on every TAB (could add caching later)

---

## Files

- `internal/shell/completer.go` - PathCompleter implementation
- `internal/shell/shell.go` - Integration via `setupPathCompletion()`
