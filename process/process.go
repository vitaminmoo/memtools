// Package process provides functionality to read memory from a process.
package process

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/memory"
	"github.com/vitaminmoo/memtools/sparsestruct"
	"golang.org/x/sys/unix"
)

// Pattern holds a value and a mask for byte-level masked searching.
type Pattern struct {
	Value []byte
	Mask  []byte
}

// Match represents a found occurrence, containing its address and the index of the pattern that it matched.
type Match struct {
	Address int64
	Map     maps.Map
}

// Scanner defines the interface for memory scanning implementations.
type Scanner interface {
	Find(ctx context.Context, r *memory.Buffer, pattern Pattern) (Match, error)
}

// Process represents a target process.
type Process struct {
	PID     int
	Scanner Scanner
	i       int64
}

// New creates a new Process with a default BruteForceScanner.
func New(pid int) *Process {
	return &Process{
		PID: pid,
	}
}

// FromName finds a process by its binary name and returns a *Process instance.
func FromName(name string) (*Process, error) {
	files, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // Not a PID
		}

		cmdlinePath := filepath.Join("/proc", file.Name(), "cmdline")
		cmdline, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue // Process might have exited, or we don't have permissions
		}

		if len(cmdline) == 0 {
			continue
		}

		args := bytes.Split(cmdline, []byte{0})
		if len(args) == 0 || len(args[0]) == 0 {
			continue
		}

		exe := filepath.Base(strings.ReplaceAll(string(args[0]), "\\", "/"))

		if exe == name {
			return New(pid), nil
		}
	}

	return nil, fmt.Errorf("process %q not found", name)
}

func (p *Process) Read(b []byte) (int, error) {
	length := len(b)
	localIov := []unix.Iovec{{Base: &b[0], Len: uint64(length)}}
	remoteIov := []unix.RemoteIovec{{Base: uintptr(p.i), Len: int(length)}}
	read, err := unix.ProcessVMReadv(p.PID, localIov, remoteIov, 0)
	if err != nil {
		return 0, fmt.Errorf("reading process memory at address 0x%x: %w", p.i, err)
	}
	p.i += int64(read)
	return read, nil
}

func (p *Process) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		p.i = offset
		return p.i, nil
	case io.SeekCurrent:
		p.i += offset
		return p.i, nil
	case io.SeekEnd:
		p.i = offset
		return p.i, nil
	default:
		return 0, fmt.Errorf("unsupported seek mode")
	}
}

// ReadStruct reads data from the process's memory at a given address into a sparsestruct.
func (p *Process) ReadStruct(addr uintptr, v any) error {
	size, err := sparsestruct.Size(v)
	if err != nil {
		return fmt.Errorf("getting size of type: %w", err)
	}
	b := make([]byte, size)
	localIov := []unix.Iovec{{Base: &b[0], Len: uint64(size)}}
	remoteIov := []unix.RemoteIovec{{Base: addr, Len: int(size)}}
	read, err := unix.ProcessVMReadv(p.PID, localIov, remoteIov, 0)
	if err != nil {
		return fmt.Errorf("reading process memory at address 0x%x: %w", addr, err)
	}
	if read != int(size) {
		return fmt.Errorf("read %d bytes, expected %d", read, size)
	}
	err = sparsestruct.Unmarshal(p, addr, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}

// Read reads data from the process's memory at a given address into a something oh god
func (p *Process) ReadUint32(addr uintptr) (uint32, error) {
	var b [4]byte
	localIov := []unix.Iovec{{Base: &b[0], Len: 4}}
	remoteIov := []unix.RemoteIovec{{Base: addr, Len: 4}}
	read, err := unix.ProcessVMReadv(p.PID, localIov, remoteIov, 0)
	if err != nil {
		return 0, fmt.Errorf("reading process memory at address 0x%x: %w", addr, err)
	}
	if read != 4 {
		return 0, fmt.Errorf("read %d bytes, expected 4", read)
	}
	return binary.LittleEndian.Uint32(b[0:]), nil
}

// Read reads data from the process's memory at a given address into a something oh god
func (p *Process) ReadUintptr(addr uintptr) (uintptr, error) {
	var b [8]byte
	localIov := []unix.Iovec{{Base: &b[0], Len: 8}}
	remoteIov := []unix.RemoteIovec{{Base: addr, Len: 8}}
	read, err := unix.ProcessVMReadv(p.PID, localIov, remoteIov, 0)
	if err != nil {
		return 0, fmt.Errorf("reading process memory at address 0x%x: %w", addr, err)
	}
	if read != 8 {
		return 0, fmt.Errorf("read %d bytes, expected 8", read)
	}
	return uintptr(binary.LittleEndian.Uint64(b[0:])), nil
}
