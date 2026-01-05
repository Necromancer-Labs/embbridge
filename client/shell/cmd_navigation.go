/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * Navigation commands: ls, cd, pwd, cat
 */

package shell

import (
	"fmt"
	"time"
)

func (m *EDBModule) doLs(path string) {
	resp, err := m.proto.Ls(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	entries, ok := resp.Data["entries"].([]interface{})
	if !ok {
		fmt.Println("Error: invalid response")
		return
	}

	for _, e := range entries {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string)
		size := toInt64(entry["size"])
		mode := toInt64(entry["mode"])
		owner, _ := entry["owner"].(string)
		mtime := toInt64(entry["mtime"])

		// Type character
		var typeChar string
		switch entryType {
		case "dir":
			typeChar = "d"
		case "link":
			typeChar = "l"
		default:
			typeChar = "."
		}

		// Format size (- for directories)
		var sizeStr string
		if entryType == "dir" {
			sizeStr = "-"
		} else {
			sizeStr = formatSizeShort(size)
		}

		// Format owner
		if owner == "" {
			owner = "-"
		}

		// Format time
		var timeStr string
		if mtime > 0 {
			t := time.Unix(mtime, 0)
			now := time.Now()
			if t.Year() == now.Year() {
				timeStr = t.Format("2 Jan 15:04")
			} else {
				timeStr = t.Format("2 Jan  2006")
			}
		} else {
			timeStr = "-"
		}

		fmt.Printf("%s%s %4s %-8s %12s  %s\n",
			typeChar,
			formatMode(uint64(mode)),
			sizeStr,
			owner,
			timeStr,
			name,
		)
	}
}

func (m *EDBModule) doCd(path string) {
	resp, err := m.proto.Cd(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if newPath, ok := resp.Data["path"].(string); ok {
		m.cwd = newPath
		m.updatePrompt()
	}
}

func (m *EDBModule) doPwd() {
	resp, err := m.proto.Pwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if path, ok := resp.Data["path"].(string); ok {
		fmt.Println(path)
	}
}

func (m *EDBModule) doCat(path string) {
	resp, err := m.proto.Cat(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	if content, ok := resp.Data["content"].([]byte); ok {
		fmt.Print(string(content))
		// Add newline if content doesn't end with one
		if len(content) > 0 && content[len(content)-1] != '\n' {
			fmt.Println()
		}
	}
}
