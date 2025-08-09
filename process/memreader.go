package process

import (
	"context"
	"fmt"

	"github.com/vitaminmoo/memtools/maps"
	"golang.org/x/sys/unix"
)

type ReadError struct {
	errorType readErrorType
	address   uint64
	nextValid uint64
}

func (e ReadError) Error() string {
	switch e.errorType {
	case ErrEndOfMemory:
		return fmt.Sprintf("reached end of memory at address 0x%X", e.address)
	case ErrEndOfMap:
		return fmt.Sprintf("address 0x%X is not mapped, next valid address is 0x%X", e.address, e.nextValid)
	default:
		return "unknown read error"
	}
}

type readErrorType int

const (
	ErrEndOfMemory readErrorType = iota
	ErrEndOfMap
)

type MemReaderConfig struct {
	filter func(maps.Map) bool
}

type MemReaderOption func(*MemReaderConfig)

type MemReader struct {
	pid int
	cur uint64
	c   MemReaderConfig
}

func NewMemReader(pid int, o ...MemReaderOption) *MemReader {
	m := &MemReader{
		pid: pid,
		cur: 0,
		c:   MemReaderConfig{},
	}
	for _, opt := range o {
		opt(&m.c)
	}
	return m
}

func (mr *MemReader) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	size := len(b)
	result, err := mr.ReadWithInfo(context.Background(), mr.cur, uint64(size))
	if err != nil {
		return 0, err
	}
	copy(b[0:], result.Data)
	mr.cur += uint64(size)
	return size, nil
}

func (mr *MemReader) Seek(offset int64, whence int) (int64, error) {
	m, err := maps.Read(mr.pid)
	if err != nil {
		return 0, err
	}
	var cur uint64
	switch whence {
	case 0: // SeekStart
		cur = uint64(offset)
	case 1: // SeekCurrent
		cur = uint64(int64(mr.cur) + offset)
	case 2: // SeekEnd
		cur = m.End() + uint64(offset)
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}
	_, err = m.Find(cur)
	if err != nil {
		return 0, err
	}
	mr.cur = cur
	return int64(mr.cur), nil
}

type Result struct {
	Data []byte
	Map  *maps.Map
}

func (mr *MemReader) ReadWithInfo(ctx context.Context, addr uint64, size uint64) (Result, error) {
	result := Result{
		Data: make([]byte, size),
	}
	maps, err := maps.Read(mr.pid)
	if err != nil {
		return result, fmt.Errorf("getting maps: %w", err)
	}
	for _, m := range maps {
		if !m.PermRead() {
			continue
		}
		if m.Contains(addr) {
			result.Map = &m
		}
		if mr.c.filter != nil && !mr.c.filter(m) {
			continue
		}
	}
	if result.Map == nil || (result.Map.End()-result.Map.Start() == 0) {
		next, err := maps.FindNext(mr.cur)
		if err != nil {
			return result, fmt.Errorf("finding next map: %w", err)
		}
		return result, ReadError{
			errorType: ErrEndOfMap,
			address:   addr,
			nextValid: next.Start(),
		}
	}

	if size == 0 {
		return result, nil
	}

	if addr+size > result.Map.End() {
		size = result.Map.End() - addr
	}

	localIov := []unix.Iovec{{Base: &result.Data[0], Len: uint64(size)}}
	remoteIov := []unix.RemoteIovec{{Base: uintptr(addr), Len: int(size)}}

	_, err = unix.ProcessVMReadv(mr.pid, localIov, remoteIov, 0)
	if err != nil {
		fmt.Printf("ProcessVMReadv failed at addr 0x%X size %d: %v, map: 0x%x\n", addr, size, err, result.Map.Start())
		return result, fmt.Errorf("reading memory: %w", err)
	}

	return result, nil
}
