package ptrchain

import (
	"context"
	"fmt"

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

func (p *Process) Read(ctx context.Context, base uintptr, v any) error {
	reader := memreader.New(p.PID)
	length, err := sparsestruct.Length(v)
	if err != nil {
		return fmt.Errorf("calculating required read size of v: %w", err)
	}
	res, err := reader.Read(ctx, base, length)
	if err != nil {
		return fmt.Errorf("reading memory: %w", err)
	}
	err = sparsestruct.Unmarshal(res.Data, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}
