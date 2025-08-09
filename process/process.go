// Package process provides functionality to read memory from a process.
package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vitaminmoo/memtools/sparsestruct"
)

type Process struct {
	PID int
}

func New(pid int) *Process {
	return &Process{
		PID: pid,
	}
}

func FromName(name string) (*Process, error) {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // Not a PID
		}

		cmdlinePath := filepath.Join("/proc", file.Name(), "cmdline")
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue // Process might have exited, or we don't have permissions
		}

		if len(cmdline) == 0 {
			continue
		}

		args := bytes.Split(cmdline, []byte{0})
		if len(args) == 0 || len(args[0]) == 0 {
			continue
		}

		exe := filepath.Base(strings.ReplaceAll(string(args[0]), "\\", "/"))

		if exe == name {
			return New(pid), nil
		}
	}

	return nil, fmt.Errorf("process %q not found", name)
}

func (p *Process) Read(ctx context.Context, base uint64, v any) error {
	reader := NewMemReader(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}

const findChunkSize = 4096

func (p *Process) Find(ctx context.Context, search []byte) (int64, error) {
	searchLen := len(search)
	if searchLen == 0 {
		return -1, fmt.Errorf("search pattern is empty")
	}

	m := NewMemReader(p.PID)
	buffer := make([]byte, findChunkSize+searchLen-1)
	overlap := make([]byte, 0, searchLen-1)
	_, err := m.Seek(0, io.SeekStart)
	if err != nil {
		// No readable memory maps
		return -1, fmt.Errorf("no readable memory found: %w", err)
	}

	for {
		readAddr, err := m.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1, fmt.Errorf("could not get current memory address: %w", err)
		}

		bytesRead, err := m.Read(buffer[len(overlap):])
		if err != nil {
			var readErr ReadError
			if errors.As(err, &readErr) {
				if readErr.errorType == ErrEndOfMap {
					fmt.Printf("Reached end of map at 0x%x, jumping to next valid region at 0x%x\n", readAddr, readErr.nextValid)
					// Jump to the next valid memory region.
					_, seekErr := m.Seek(int64(readErr.nextValid), io.SeekStart)
					if seekErr != nil {
						// This could happen if nextValid is invalid or we are at the end.
						return -1, fmt.Errorf("pattern not found, seek failed: %w", seekErr)
					}
					overlap = overlap[:0] // Reset overlap after a jump.
					continue
				} else if readErr.errorType == ErrEndOfMemory {
					// Reached the end of all readable memory.
					break
				}
			}
			// Some other unexpected error.
			return -1, fmt.Errorf("error reading memory at 0x%x: %w", readAddr, err)
		}

		if bytesRead == 0 {
			continue
		}

		dataToSearch := buffer[:len(overlap)+bytesRead]

		if idx := bytes.Index(dataToSearch, search); idx != -1 {
			return readAddr + int64(idx) - int64(len(overlap)), nil
		}

		if len(dataToSearch) > searchLen-1 {
			overlap = append(overlap[:0], dataToSearch[len(dataToSearch)-(searchLen-1):]...)
		} else {
			overlap = append(overlap[:0], dataToSearch...)
		}
	}

	return -1, fmt.Errorf("pattern not found")
}
