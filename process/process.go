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
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

type Pattern struct {
	Value []byte
	Mask  []byte
}

type Match struct {
	Address      int64
	PatternIndex int
}

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

func (p *Process) FindFirst(ctx context.Context, pattern Pattern) (int64, error) {
	if len(pattern.Value) == 0 {
		return -1, fmt.Errorf("search pattern is empty")
	}
	if len(pattern.Mask) > 0 && len(pattern.Value) != len(pattern.Mask) {
		return -1, fmt.Errorf("value and mask length mismatch")
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
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.findFirstWorker(ctx, pattern, readableMaps, resultChan, cancel)
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

func (p *Process) findFirstWorker(ctx context.Context, pattern Pattern, mapsChan <-chan maps.Map, resultChan chan<- int64, cancel context.CancelFunc) {
	searchLen := len(pattern.Value)
	buffer := make([]byte, findChunkSize+searchLen-1)

	for m := range mapsChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		addr, err := p.findFirstInMap(ctx, pattern, m, buffer)
		if err == nil {
			cancel()
			resultChan <- addr
			return
		}
	}
}

func (p *Process) findFirstInMap(ctx context.Context, pattern Pattern, m maps.Map, buffer []byte) (int64, error) {
	searchLen := len(pattern.Value)
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
		if idx := findMasked(dataToSearch, pattern); idx != -1 {
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

func (p *Process) Find(ctx context.Context, patterns []Pattern) ([]Match, error) {
	for i, pattern := range patterns {
		if len(pattern.Value) == 0 {
			return nil, fmt.Errorf("pattern %d is empty", i)
		}
		if len(pattern.Mask) > 0 && len(pattern.Value) != len(pattern.Mask) {
			return nil, fmt.Errorf("value and mask length mismatch for pattern %d", i)
		}
	}

	allMaps, err := maps.Read(p.PID)
	if err != nil {
		return nil, fmt.Errorf("could not read memory maps: %w", err)
	}

	readableMaps := make(chan maps.Map, len(allMaps))
	for _, m := range allMaps {
		if m.PermRead() {
			readableMaps <- m
		}
	}
	close(readableMaps)

	var wg sync.WaitGroup
	resultChan := make(chan []Match)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.findAllWorker(ctx, patterns, readableMaps, resultChan)
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allMatches []Match
	for matches := range resultChan {
		allMatches = append(allMatches, matches...)
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Address < allMatches[j].Address
	})

	return allMatches, nil
}

func (p *Process) findAllWorker(ctx context.Context, patterns []Pattern, mapsChan <-chan maps.Map, resultChan chan<- []Match) {
	maxPatternLen := 0
	for _, p := range patterns {
		if len(p.Value) > maxPatternLen {
			maxPatternLen = len(p.Value)
		}
	}
	buffer := make([]byte, findChunkSize+maxPatternLen-1)

	for m := range mapsChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		matches, err := p.findAllInMap(ctx, patterns, m, buffer)
		if err == nil && len(matches) > 0 {
			resultChan <- matches
		}
	}
}

func (p *Process) findAllInMap(ctx context.Context, patterns []Pattern, m maps.Map, buffer []byte) ([]Match, error) {
	var matches []Match
	maxPatternLen := 0
	for _, p := range patterns {
		if len(p.Value) > maxPatternLen {
			maxPatternLen = len(p.Value)
		}
	}

	overlap := make([]byte, 0, maxPatternLen-1)
	currentAddr := m.Start()
	memReader := NewMemReader(p.PID)

	for currentAddr < m.End() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, err := memReader.Seek(int64(currentAddr), io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("seek failed in map %s: %w", m.PathName(), err)
		}

		readStartAddr, err := memReader.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
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
			return nil, fmt.Errorf("read failed in map %s: %w", m.PathName(), err)
		}

		if bytesRead == 0 {
			currentAddr++
			continue
		}

		dataToSearch := buffer[:len(overlap)+bytesRead]
		for i, pattern := range patterns {
			searchLen := len(pattern.Value)
			for j := 0; j <= len(dataToSearch)-searchLen; j++ {
				if findMasked(dataToSearch[j:j+searchLen], pattern) == 0 {
					matchAddr := readStartAddr + int64(j) - int64(len(overlap))
					matches = append(matches, Match{Address: matchAddr, PatternIndex: i})
				}
			}
		}

		currentAddr = uint64(readStartAddr + int64(bytesRead))

		if len(dataToSearch) > maxPatternLen-1 {
			overlap = append(overlap[:0], dataToSearch[len(dataToSearch)-(maxPatternLen-1):]...)
		} else {
			overlap = append(overlap[:0], dataToSearch...)
		}
	}

	return matches, nil
}

func findMasked(data []byte, pattern Pattern) int {
	if len(pattern.Mask) == 0 {
		return bytes.Index(data, pattern.Value)
	}

	patternLen := len(pattern.Value)
	for i := 0; i <= len(data)-patternLen; i++ {
		found := true
		for j := range patternLen {
			if (data[i+j] & pattern.Mask[j]) != (pattern.Value[j] & pattern.Mask[j]) {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
