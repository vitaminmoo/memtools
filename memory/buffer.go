// Package memory provides utilities for interacting with process memory.
package memory

import (
	"io"

	"golang.org/x/sys/unix"
)

func NewBuffer(pid int, start, end uintptr, bufLen int) *Buffer {
	if bufLen <= 0 {
		bufLen = 1024 * 1024 * 1024
	}
	return &Buffer{
		pid:   pid,
		start: start,
		addr:  start,
		end:   end,
		data:  make([]byte, bufLen),
	}
}

type Buffer struct {
	pid      int
	start    uintptr
	addr     uintptr
	end      uintptr
	data     []byte
	i        int
	err      error
	syscalls int
}

func (b *Buffer) Reset(addr uintptr) {
	b.addr = addr
	// b.data = b.data[:0]
	//b.data = make([]byte, cap(b.data))
	for i := range b.data {
		b.data[i] = 0
	}
	b.i = 0
	b.err = nil
	b.syscalls = 0
}

func (b *Buffer) Next(l int) ([]byte, error) {
	if b.i+l > len(b.data) || b.syscalls == 0 {
		// Asking for more data than we have. refill
		if err := b.Refill(); err != nil {
			return nil, err
		}
	}

	b.i += l
	return b.data[b.i-l : b.i], nil
}

// Peek allows direct access to the current remaining buffer
func (b *Buffer) Peek() []byte {
	return b.data[b.i:]
}

func (b *Buffer) Rewind(l int) {
	if l > b.i {
		b.i = 0
	} else {
		b.i -= l
	}
}

// Discard consumes data in the current buffer
func (b *Buffer) Discard(n int) {
	b.i += n
}

// Refill forces the buffer to try to put at least one more byte into its buffer
func (b *Buffer) Refill() error {
	if b.err != nil {
		// We already know we can't get more data
		return b.err
	}
	var n int
	if b.i != 0 {
		// shift existing data down over the read portion of the buffer
		n = copy(b.data[:cap(b.data)], b.data[b.i:])
		b.i = 0
	}

	remaining := int(b.end - b.addr)
	if remaining == 0 {
		b.err = io.EOF
		return b.err
	}
	toRead := min(cap(b.data)-n, remaining)

	localIov := []unix.Iovec{{Base: &b.data[n], Len: uint64(toRead)}}
	remoteIov := []unix.RemoteIovec{{Base: b.addr, Len: int(toRead)}}
	read, err := unix.ProcessVMReadv(b.pid, localIov, remoteIov, 0)
	if err != nil {
		b.err = err
		return b.err
	}
	b.syscalls++
	b.addr += uintptr(read)
	// zero out the rest of the slice
	// Can't use this because it decreases capacity
	// b.data = b.data[:n+read]
	for i := n + read; i < cap(b.data); i++ {
		b.data[i] = 0
	}
	return nil
}
