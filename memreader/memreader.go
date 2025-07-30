package memreader

import (
	"context"
	"fmt"

	"github.com/vitaminmoo/memtools/pidmaps"
	"golang.org/x/sys/unix"
)

type Config struct {
	filter func(pidmaps.Map) bool
}

type Option func(*Config)

type MemReader struct {
	pid int
	c   Config
}

func New(pid int, o ...Option) *MemReader {
	m := &MemReader{
		pid: pid,
		c:   Config{},
	}
	for _, opt := range o {
		opt(&m.c)
	}
	return m
}

type Result struct {
	Data []byte
	Map  *pidmaps.Map
}

func (mr *MemReader) Read(ctx context.Context, addr uintptr, size uint64) (Result, error) {
	result := Result{
		Data: make([]byte, size),
	}
	maps, err := pidmaps.Maps(mr.pid)
	if err != nil {
		return result, fmt.Errorf("getting maps: %w", err)
	}
	for _, m := range maps {
		if m.Contains(addr) {
			result.Map = &m
		}
		if mr.c.filter != nil && !mr.c.filter(m) {
			continue
		}
	}
	if result.Map == nil {
		return result, fmt.Errorf("address 0x%X not within pid %d's maps", addr, mr.pid)
	}

	if size == 0 {
		return result, nil
	}

	localIov := []unix.Iovec{{Base: &result.Data[0], Len: uint64(size)}}
	remoteIov := []unix.RemoteIovec{{Base: addr, Len: int(size)}}

	_, err = unix.ProcessVMReadv(mr.pid, localIov, remoteIov, 0)
	if err != nil {
		return result, fmt.Errorf("reading memory: %w", err)
	}

	return result, nil
}
