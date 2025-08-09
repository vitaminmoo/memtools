// Package process provides functionality to read memory from a process.
package process

import (
	"bufio"
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
	ix := 0
	m := NewMemReader(p.PID)
	r := bufio.NewReader(m)
	offset := int64(0)
	for ix < len(search) {
		b, err := r.ReadByte()
		if err != nil {
			var readErr ReadError
			if ok := errors.As(err, &readErr); ok {
				switch readErr.errorType {
				case ErrEndOfMap:
					fmt.Printf("Skipping unreadable memory region at offset %d, next valid at %d\n", offset, readErr.nextValid)
					// Skip unreadable memory regions
					offset = int64(readErr.nextValid)
					m.Seek(offset, io.SeekStart)
					ix = 0
					continue
				case ErrEndOfMemory:
					return -1, fmt.Errorf("reached end of memory while searching for pattern")
				}
			}
			return -1, fmt.Errorf("reading byte at offset %d: %w", offset, err)
		}
		if search[ix] == b {
			ix++
		} else {
			ix = 0
		}
		offset++
	}
	m.Seek(offset-int64(len(search)), 0) // Seeks to the beginning of the searched []byte
	return offset - int64(len(search)), nil
}
