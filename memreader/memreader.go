package memreader

import (
	"fmt"

	"github.com/vitaminmoo/memtools/pidmaps"
	"golang.org/x/sys/unix"
)

type MemReader struct {
	pid int
}

func NewMemReader(pid int) *MemReader {
	return &MemReader{
		pid: pid,
	}
}

type Result struct {
	Data []byte
	Map  pidmaps.Map
}

func (mr *MemReader) Read(addr uintptr, size int) (Result, error) {
	result := Result{
		Data: make([]byte, size),
	}
	maps, err := pidmaps.Maps(mr.pid)
	if err != nil {
		return result, fmt.Errorf("getting maps: %w", err)
	}
	for _, m := range maps {
		if m.Contains(addr) {
			result.Map = m
		}
	}
	buf := make([]byte, size)
	if size == 0 {
		return result, nil
	}

	localIov := []unix.Iovec{{Base: &buf[0], Len: uint64(size)}}
	remoteIov := []unix.RemoteIovec{{Base: addr, Len: size}}

	_, err = unix.ProcessVMReadv(mr.pid, localIov, remoteIov, 0)
	if err != nil {
		return result, fmt.Errorf("reading memory: %w", err)
	}

	return result, nil
}
