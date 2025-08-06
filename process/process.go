// Package process provides functionality to read memory from a process.
package process

import (
	"context"
	"fmt"
	"io"

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

func (p *Process) Read(ctx context.Context, base uint64, v any) error {
	reader := NewMemReader(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}
