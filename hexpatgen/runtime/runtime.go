// Package runtime provides types imported by code generated from hexpatgen.
package runtime

import (
	"fmt"
	"io"
	"strings"
)

// ReadContext wraps an io.ReadSeeker with cycle detection state.
type ReadContext struct {
	r       io.ReadSeeker
	visited map[uintptr]bool
}

// NewReadContext creates a new ReadContext wrapping the given io.ReadSeeker.
func NewReadContext(r io.ReadSeeker) *ReadContext {
	return &ReadContext{
		r:       r,
		visited: make(map[uintptr]bool),
	}
}

// ReadAt seeks to addr and reads exactly len(buf) bytes.
func (c *ReadContext) ReadAt(buf []byte, addr int64) (int, error) {
	if _, err := c.r.Seek(addr, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(c.r, buf)
}

// Visit returns true if addr was already visited, marking it if not.
func (c *ReadContext) Visit(addr uintptr) bool {
	if c.visited[addr] {
		return true
	}
	c.visited[addr] = true
	return false
}

// ChainError represents a field-level read error with path and address context.
type ChainError struct {
	Path    string
	Address uintptr
	Err     error
}

func (e ChainError) Error() string {
	return fmt.Sprintf("%s @ 0x%x: %v", e.Path, e.Address, e.Err)
}

func (e ChainError) Unwrap() error {
	return e.Err
}

// Errors collects field-level read errors.
type Errors []ChainError

// Add appends a new error.
func (e *Errors) Add(path string, addr uintptr, err error) {
	*e = append(*e, ChainError{Path: path, Address: addr, Err: err})
}

// HasFatal returns true if any errors were recorded.
func (e Errors) HasFatal() bool {
	return len(e) > 0
}

// Error returns a summary of all errors.
func (e Errors) Error() string {
	if len(e) == 0 {
		return "<no errors>"
	}
	var b strings.Builder
	for i, ce := range e {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(ce.Error())
	}
	return b.String()
}
