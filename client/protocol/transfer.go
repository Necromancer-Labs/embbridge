/*
 * embbridge - Embedded Debug Bridge
 * https://github.com/Necromancer-Labs/embbridge
 *
 * File transfer - chunked pull/push with progress reporting
 */

package protocol

import "fmt"

// DataMsg is a chunk of data for file transfer
type DataMsg struct {
	Type string `msgpack:"type"`
	ID   uint32 `msgpack:"id"`
	Seq  uint32 `msgpack:"seq"`
	Data []byte `msgpack:"data"`
	Done bool   `msgpack:"done"`
}

// TransferProgress is called during file transfers with progress info
type TransferProgress func(transferred, total int64)

// Pull downloads a file from the device with progress reporting
func (p *Protocol) Pull(remotePath string, progress TransferProgress) ([]byte, int64, uint32, error) {
	args := map[string]any{"path": remotePath}
	if _, err := p.SendRequest("pull", args); err != nil {
		return nil, 0, 0, err
	}

	// Receive initial response with file info
	resp, err := p.RecvResponse()
	if err != nil {
		return nil, 0, 0, err
	}

	if !resp.OK {
		return nil, 0, 0, fmt.Errorf("%s", resp.Error)
	}

	// Extract file size and mode
	size := toInt64(resp.Data["size"])
	mode := toInt64(resp.Data["mode"])

	// Receive data chunks
	var data []byte
	var transferred int64

	for {
		var chunk DataMsg
		if err := p.Recv(&chunk); err != nil {
			return nil, 0, 0, fmt.Errorf("receive chunk: %w", err)
		}

		if chunk.Type != "data" {
			return nil, 0, 0, fmt.Errorf("expected data, got %s", chunk.Type)
		}

		data = append(data, chunk.Data...)
		transferred += int64(len(chunk.Data))

		if progress != nil {
			progress(transferred, size)
		}

		if chunk.Done {
			break
		}
	}

	return data, size, uint32(mode), nil
}

// SendData sends a data chunk
func (p *Protocol) SendData(id, seq uint32, data []byte, done bool) error {
	msg := DataMsg{
		Type: "data",
		ID:   id,
		Seq:  seq,
		Data: data,
		Done: done,
	}
	return p.Send(msg)
}

// Push uploads a file to the device with progress reporting
func (p *Protocol) Push(remotePath string, data []byte, mode uint32, progress TransferProgress) error {
	args := map[string]any{
		"path": remotePath,
		"size": uint64(len(data)),
		"mode": uint64(mode),
	}
	id, err := p.SendRequest("push", args)
	if err != nil {
		return err
	}

	// Receive OK response
	resp, err := p.RecvResponse()
	if err != nil {
		return err
	}

	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}

	// Send data in chunks
	total := int64(len(data))
	var seq uint32
	var transferred int64

	for transferred < total {
		end := min(transferred+DefaultChunk, total)
		chunk := data[transferred:end]
		done := (end >= total)

		if err := p.SendData(id, seq, chunk, done); err != nil {
			return fmt.Errorf("send chunk: %w", err)
		}

		transferred = end
		seq++

		if progress != nil {
			progress(transferred, total)
		}
	}

	return nil
}
