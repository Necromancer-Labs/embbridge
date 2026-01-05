/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File operation commands: rm, mv, cp, mkdir, chmod
 */

package shell

import (
	"fmt"
)

func (m *EDBModule) doRm(path string) {
	resp, err := m.proto.Rm(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Printf("Removed: %s\n", path)
}

func (m *EDBModule) doMv(src, dst string) {
	resp, err := m.proto.Mv(src, dst)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Printf("Moved: %s -> %s\n", src, dst)
}

func (m *EDBModule) doCp(src, dst string) {
	resp, err := m.proto.Cp(src, dst)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Printf("Copied: %s -> %s\n", src, dst)
}

func (m *EDBModule) doMkdir(path string) {
	resp, err := m.proto.Mkdir(path, 0755)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Printf("Created: %s\n", path)
}

func (m *EDBModule) doChmod(modeStr, path string) {
	// Parse octal mode string (e.g., "755", "0644")
	var mode uint32
	_, err := fmt.Sscanf(modeStr, "%o", &mode)
	if err != nil {
		fmt.Printf("Error: invalid mode '%s' (use octal, e.g., 755)\n", modeStr)
		return
	}

	resp, err := m.proto.Chmod(path, mode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !resp.OK {
		fmt.Printf("Error: %s\n", resp.Error)
		return
	}

	fmt.Printf("Changed mode: %s -> %o\n", path, mode)
}
