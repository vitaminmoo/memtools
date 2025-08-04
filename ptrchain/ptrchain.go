package ptrchain

import (
	"context"
	"fmt"
	"io"

	"github.com/vitaminmoo/memtools/memreader"
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
	reader := memreader.New(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}
