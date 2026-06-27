// Package runtime provides types imported by code generated from hexpatgen.
package runtime

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// MaxDynamicArrayLen bounds the element count of a dynamically-sized array
// decoded from remote memory. A C++ vector whose Begin/End pointers are stale,
// or torn from being read while the target mutates it, can yield an absurd
// length (e.g. a uint32 pointer subtraction that underflows to ~4 billion).
// Allocating and reading that many elements would hang the reader and exhaust
// memory, so any computed length above this is treated as corrupt.
const MaxDynamicArrayLen = 1 << 20

// ErrArrayTooLong is recorded against a field when its dynamically-computed
// array length exceeds MaxDynamicArrayLen and is therefore assumed corrupt.
var ErrArrayTooLong = errors.New("dynamic array length exceeds MaxDynamicArrayLen; treating as corrupt")

// BoundedLen validates a dynamically-computed array length. It returns the
// length and true when 0 <= n <= MaxDynamicArrayLen, or (0, false) when the
// value is negative or implausibly large and should be treated as corrupt.
func BoundedLen(n int) (int, bool) {
	if n < 0 || n > MaxDynamicArrayLen {
		return 0, false
	}
	return n, true
}

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

// ReadReq describes one address range to fetch into Buf.
type ReadReq struct {
	Addr uintptr
	Buf  []byte
}

// BatchReader is implemented by ReadSeekers that can satisfy multiple
// ReadReqs in a single underlying operation (e.g. one process_vm_readv
// syscall with multiple iovecs).
type BatchReader interface {
	ReadBatch(reqs []ReadReq) error
}

// ReadBatch fills every req.Buf with bytes from req.Addr. If the underlying
// io.ReadSeeker implements BatchReader, the whole batch is issued in a single
// underlying call; otherwise reads are performed sequentially via ReadAt.
func (c *ReadContext) ReadBatch(reqs []ReadReq) error {
	if len(reqs) == 0 {
		return nil
	}
	if br, ok := c.r.(BatchReader); ok {
		return br.ReadBatch(reqs)
	}
	for i := range reqs {
		if _, err := c.ReadAt(reqs[i].Buf, int64(reqs[i].Addr)); err != nil {
			return err
		}
	}
	return nil
}

// Collector queues read requests and flushes them in one batch. Use this when
// a hot path needs many small reads that can be issued together (e.g. an array
// of pointer dereferences). Each Add returns the buffer the caller will own
// after Flush; reading from the buffer before Flush has undefined contents.
type Collector struct {
	ctx  *ReadContext
	reqs []ReadReq
}

// NewCollector returns a Collector bound to ctx.
func NewCollector(ctx *ReadContext) *Collector {
	return &Collector{ctx: ctx}
}

// Add enqueues a read of size bytes at addr and returns the destination
// buffer. The buffer is valid to read after Flush returns.
func (c *Collector) Add(addr uintptr, size int) []byte {
	buf := make([]byte, size)
	c.reqs = append(c.reqs, ReadReq{Addr: addr, Buf: buf})
	return buf
}

// Len returns the number of queued requests.
func (c *Collector) Len() int { return len(c.reqs) }

// Flush issues all queued requests. After Flush, each buffer returned by Add
// is populated with the bytes read from its address. The collector is reset
// and may be reused.
func (c *Collector) Flush() error {
	if len(c.reqs) == 0 {
		return nil
	}
	err := c.ctx.ReadBatch(c.reqs)
	c.reqs = c.reqs[:0]
	return err
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
