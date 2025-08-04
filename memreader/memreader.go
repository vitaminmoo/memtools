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

type MemReadSeeker struct {
	pid int
	cur uint64
	c   Config
}

func New(pid int, o ...Option) *MemReadSeeker {
	m := &MemReadSeeker{
		pid: pid,
		cur: 0,
		c:   Config{},
	}
	for _, opt := range o {
		opt(&m.c)
	}
	return m
}

func (mr *MemReadSeeker) Read(b []byte) (int, error) {
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

func (mr *MemReadSeeker) Seek(offset int64, whence int) (int64, error) {
	m, err := pidmaps.Read(mr.pid)
	if err != nil {
		return 0, err
	}
	cur := mr.cur
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
	Map  *pidmaps.Map
}

func (mr *MemReadSeeker) ReadWithInfo(ctx context.Context, addr uint64, size uint64) (Result, error) {
	fmt.Printf("addr: %X, size: %d\n", addr, size)
	result := Result{
		Data: make([]byte, size),
	}
	maps, err := pidmaps.Read(mr.pid)
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
	remoteIov := []unix.RemoteIovec{{Base: uintptr(addr), Len: int(size)}}

	_, err = unix.ProcessVMReadv(mr.pid, localIov, remoteIov, 0)
	if err != nil {
		return result, fmt.Errorf("reading memory: %w", err)
	}

	return result, nil
}
