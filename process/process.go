// Package process provides functionality to read memory from a process.
package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

type Process struct {
	PID int
}

func New(pid int) *Process {
	return &Process{
		PID: pid,
	}
}

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

func (p *Process) Read(ctx context.Context, base uint64, v any) error {
	reader := NewMemReader(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}

const findChunkSize = 4096

func (p *Process) Find(ctx context.Context, search []byte) (int64, error) {
	searchLen := len(search)
	if searchLen == 0 {
		return -1, fmt.Errorf("search pattern is empty")
	}

	allMaps, err := maps.Read(p.PID)
	if err != nil {
		return -1, fmt.Errorf("could not read memory maps: %w", err)
	}

	readableMaps := make(chan maps.Map, len(allMaps))
	for _, m := range allMaps {
		if m.PermRead() {
			readableMaps <- m
		}
	}
	close(readableMaps)

	var wg sync.WaitGroup
	resultChan := make(chan int64, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.findWorker(ctx, search, readableMaps, resultChan, cancel)
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	if addr, ok := <-resultChan; ok {
		return addr, nil
	}

	return -1, fmt.Errorf("pattern not found")
}

func (p *Process) findWorker(ctx context.Context, search []byte, mapsChan <-chan maps.Map, resultChan chan<- int64, cancel context.CancelFunc) {
	searchLen := len(search)
	buffer := make([]byte, findChunkSize+searchLen-1)

	for m := range mapsChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		addr, err := p.findInMap(ctx, search, m, buffer)
		if err == nil {
			cancel()
			resultChan <- addr
			return
		}
	}
}

func (p *Process) findInMap(ctx context.Context, search []byte, m maps.Map, buffer []byte) (int64, error) {
	searchLen := len(search)
	overlap := make([]byte, 0, searchLen-1)
	currentAddr := m.Start()
	memReader := NewMemReader(p.PID)

	for currentAddr < m.End() {
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		default:
		}

		_, err := memReader.Seek(int64(currentAddr), io.SeekStart)
		if err != nil {
			return -1, fmt.Errorf("seek failed in map %s: %w", m.PathName(), err)
		}

		readStartAddr, err := memReader.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1, err
		}

		bytesRead, err := memReader.Read(buffer[len(overlap):])
		if err != nil {
			var readErr ReadError
			if errors.As(err, &readErr) {
				if readErr.errorType == ErrEndOfMap {
					currentAddr = readErr.nextValid
					overlap = overlap[:0]
					continue
				}
			}
			return -1, fmt.Errorf("read failed in map %s: %w", m.PathName(), err)
		}

		if bytesRead == 0 {
			currentAddr++
			continue
		}

		dataToSearch := buffer[:len(overlap)+bytesRead]
		if idx := bytes.Index(dataToSearch, search); idx != -1 {
			return readStartAddr + int64(idx) - int64(len(overlap)), nil
		}

		currentAddr = uint64(readStartAddr + int64(bytesRead))

		if len(dataToSearch) > searchLen-1 {
			overlap = append(overlap[:0], dataToSearch[len(dataToSearch)-(searchLen-1):]...)
		} else {
			overlap = append(overlap[:0], dataToSearch...)
		}
	}

	return -1, fmt.Errorf("pattern not found in map")
}
