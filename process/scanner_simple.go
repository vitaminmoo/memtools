package process

import (
	"context"
	"fmt"

	"github.com/vitaminmoo/memtools/memory"
)

// SimpleScanner implements the Scanner interface with a basic, non-parallel approach.
type SimpleScanner struct{}

func (s *SimpleScanner) Find(ctx context.Context, buffer *memory.Buffer, pattern Pattern) (Match, error) {
	if len(pattern.Value) == 0 {
		return Match{}, fmt.Errorf("pattern is empty")
	}
	if len(pattern.Mask) > 0 && len(pattern.Value) != len(pattern.Mask) {
		return Match{}, fmt.Errorf("value and mask length mismatch for pattern")
	}

	// implement a scan of r for pattern.Value with optional pattern.Mask

	return Match{}, nil
}
