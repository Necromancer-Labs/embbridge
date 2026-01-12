/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File transfer commands: pull, push
 */

package shell

import (
	"fmt"
	"os"
	"time"
)

func (m *EDBModule) doGet(remotePath, localPath string) {
	fmt.Printf("↓ Downloading %s...\n", remotePath)
	startTime := time.Now()

	// Progress callback
	var lastPrint time.Time
	progress := func(transferred, total int64) {
		// Rate limit progress output
		if time.Since(lastPrint) > 100*time.Millisecond {
			percent := float64(transferred) / float64(total) * 100
			fmt.Printf("\r  %s / %s (%.1f%%)", formatBytes(transferred), formatBytes(total), percent)
			lastPrint = time.Now()
		}
	}

	data, _, mode, err := m.proto.Pull(remotePath, progress)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	// Write to local file (create or truncate)
	f, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		fmt.Printf("\nError creating file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		fmt.Printf("\nError writing file: %v\n", err)
		return
	}

	elapsed := time.Since(startTime)
	speed := float64(len(data)) / elapsed.Seconds()
	fmt.Printf("\r  %s downloaded in %v (%s/s)\n", formatBytes(int64(len(data))), elapsed.Round(time.Millisecond), formatBytes(int64(speed)))
}

func (m *EDBModule) doPut(localPath, remotePath string) {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get file mode
	info, err := os.Stat(localPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mode := uint32(info.Mode().Perm())

	fmt.Printf("↑ Uploading %s (%s)...\n", localPath, formatBytes(int64(len(data))))
	startTime := time.Now()

	// Progress callback
	var lastPrint time.Time
	progress := func(transferred, total int64) {
		if time.Since(lastPrint) > 100*time.Millisecond {
			percent := float64(transferred) / float64(total) * 100
			fmt.Printf("\r  %s / %s (%.1f%%)", formatBytes(transferred), formatBytes(total), percent)
			lastPrint = time.Now()
		}
	}

	if err := m.proto.Push(remotePath, data, mode, progress); err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	elapsed := time.Since(startTime)
	speed := float64(len(data)) / elapsed.Seconds()
	fmt.Printf("\r  %s uploaded in %v (%s/s)\n", formatBytes(int64(len(data))), elapsed.Round(time.Millisecond), formatBytes(int64(speed)))
}
