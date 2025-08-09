// Package process provides functionality to read memory from a process.
package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

// Pattern holds a value and a mask for byte-level masked searching.
type Pattern struct {
	Value []byte
	Mask  []byte
}

// Match represents a found occurrence, containing its address and the index of the pattern that it matched.
type Match struct {
	Address      int64
	PatternIndex int
	Map          maps.Map
}

// Scanner defines the interface for memory scanning implementations.
type Scanner interface {
	Find(ctx context.Context, p *Process, patterns []Pattern) ([]Match, error)
}

// Process represents a target process.
type Process struct {
	PID     int
	Scanner Scanner
}

// New creates a new Process with a default BruteForceScanner.
func New(pid int) *Process {
	return &Process{
		PID:     pid,
		Scanner: &BruteForceScanner{},
	}
}

// FromName finds a process by its binary name and returns a *Process instance.
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

		exe := filepath.Base(strings.ReplaceAll(string(args[0]), "\\\\", "/"))

		if exe == name {
			return New(pid), nil
		}
	}

	return nil, fmt.Errorf("process %q not found", name)
}

// Read reads data from the process's memory at a given base address into a struct.
func (p *Process) Read(ctx context.Context, base uint64, v any) error {
	reader := NewMemReader(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}

// Find delegates to the configured scanner to find all occurrences of multiple patterns.
func (p *Process) Find(ctx context.Context, patterns []Pattern) ([]Match, error) {
	return p.Scanner.Find(ctx, p, patterns)
}
