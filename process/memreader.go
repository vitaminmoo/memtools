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
	pid        int
	cur        uint64
	c          MemReaderConfig
	maps       maps.Maps
	currentMap *maps.Map
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

func (mr *MemReader) refreshMaps() error {
	maps, err := maps.Read(mr.pid)
	if err != nil {
		return fmt.Errorf("reading maps for pid %d: %w", mr.pid, err)
	}
	mr.maps = maps
	mr.currentMap = nil
	return nil
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
	n := copy(b, result.Data)
	mr.cur += uint64(n)
	return n, err
}

func (mr *MemReader) Seek(offset int64, whence int) (int64, error) {
	if mr.maps == nil {
		if err := mr.refreshMaps(); err != nil {
			return 0, err
		}
	}

	var cur uint64
	switch whence {
	case 0: // SeekStart
		cur = uint64(offset)
	case 1: // SeekCurrent
		cur = uint64(int64(mr.cur) + offset)
	case 2: // SeekEnd
		cur = mr.maps.End() + uint64(offset)
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}

	m, err := mr.maps.Find(cur)
	if err != nil {
		if err := mr.refreshMaps(); err != nil {
			return 0, err
		}
		m, err = mr.maps.Find(cur)
		if err != nil {
			next, findErr := mr.maps.FindNext(cur)
			if findErr != nil {
				return 0, err
			}
			cur = next.Start()
			mr.currentMap = &next
		} else {
			mr.currentMap = &m
		}
	} else {
		mr.currentMap = &m
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

	findInMaps := func(maps maps.Maps) *maps.Map {
		for i := range maps {
			if maps[i].Contains(addr) && maps[i].PermRead() {
				if mr.c.filter == nil || mr.c.filter(maps[i]) {
					return &maps[i]
				}
			}
		}
		return nil
	}

	if mr.currentMap == nil || !mr.currentMap.Contains(addr) || !mr.currentMap.PermRead() || (mr.c.filter != nil && !mr.c.filter(*mr.currentMap)) {
		var m *maps.Map
		if mr.maps != nil {
			m = findInMaps(mr.maps)
		}

		if m == nil {
			if err := mr.refreshMaps(); err != nil {
				return result, err
			}
			m = findInMaps(mr.maps)
		}

		if m == nil {
			next, err := mr.maps.FindNext(addr)
			if err != nil {
				if mr.maps.End() <= addr {
					return result, ReadError{errorType: ErrEndOfMemory, address: addr}
				}
				return result, fmt.Errorf("finding next map: %w", err)
			}
			return result, ReadError{
				errorType: ErrEndOfMap,
				address:   addr,
				nextValid: next.Start(),
			}
		}
		mr.currentMap = m
	}
	result.Map = mr.currentMap

	if size == 0 {
		return result, nil
	}

	if addr+size > result.Map.End() {
		size = result.Map.End() - addr
	}
	result.Data = result.Data[:size]

	localIov := []unix.Iovec{{Base: &result.Data[0], Len: uint64(size)}}
	remoteIov := []unix.RemoteIovec{{Base: uintptr(addr), Len: int(size)}}

	n, err := unix.ProcessVMReadv(mr.pid, localIov, remoteIov, 0)
	if err != nil {
		mr.currentMap = nil // Invalidate map on error
		next, findErr := mr.maps.FindNext(addr)
		if findErr != nil {
			return result, fmt.Errorf("finding next map after read error at 0x%X: %w", addr, findErr)
		}
		return result, ReadError{
			errorType: ErrEndOfMap,
			address:   addr,
			nextValid: next.Start(),
		}
	}
	result.Data = result.Data[:n]

	return result, nil
}
