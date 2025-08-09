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

// Pattern holds a value and a mask for byte-level masked searching.
type Pattern struct {
	Value []byte
	Mask  []byte
}

// Match represents a found occurrence, containing its address and the index of the pattern that it matched.
type Match struct {
	Address      int64
	PatternIndex int
	Map          maps.Map
}

// Scanner defines the interface for memory scanning implementations.
type Scanner interface {
	Find(ctx context.Context, p *Process, patterns []Pattern) ([]Match, error)
}

// Process represents a target process.
type Process struct {
	PID     int
	Scanner Scanner
}

// New creates a new Process with a default BruteForceScanner.
func New(pid int) *Process {
	return &Process{
		PID:     pid,
		Scanner: &BruteForceScanner{},
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

		exe := filepath.Base(strings.ReplaceAll(string(args[0]), "\\\\", "/"))

		if exe == name {
			return New(pid), nil
		}
	}

	return nil, fmt.Errorf("process %q not found", name)
}

// Read reads data from the process's memory at a given base address into a struct.
func (p *Process) Read(ctx context.Context, base uint64, v any) error {
	reader := NewMemReader(p.PID)
	reader.Seek(int64(base), io.SeekStart)
	err := sparsestruct.Unmarshal(reader, v)
	if err != nil {
		return fmt.Errorf("unmarshalling sparse struct: %w", err)
	}
	return nil
}

// Find delegates to the configured scanner to find all occurrences of multiple patterns.
func (p *Process) Find(ctx context.Context, patterns []Pattern) ([]Match, error) {
	return p.Scanner.Find(ctx, p, patterns)
}

const findChunkSize = 4096

// BruteForceScanner implements the Scanner interface using a parallelized brute-force approach.
type BruteForceScanner struct{}

func (s *BruteForceScanner) Find(ctx context.Context, p *Process, patterns []Pattern) ([]Match, error) {
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
			s.findAllWorker(ctx, p, patterns, readableMaps, resultChan)
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for i, pattern := range patterns {
		if len(pattern.Value) == 0 {
			return nil, fmt.Errorf("pattern %d is empty", i)
		}
		if len(pattern.Mask) > 0 && len(pattern.Value) != len(pattern.Mask) {
			return nil, fmt.Errorf("value and mask length mismatch for pattern %d", i)
		}
	}

	var allMatches []Match
	for matches := range resultChan {
		allMatches = append(allMatches, matches...)
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Address < allMatches[j].Address
	})

	return allMatches, nil
}

func (s *BruteForceScanner) findAllWorker(ctx context.Context, p *Process, patterns []Pattern, mapsChan <-chan maps.Map, resultChan chan<- []Match) {
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

		matches, err := s.findAllInMap(ctx, p, patterns, m, buffer)
		if err == nil && len(matches) > 0 {
			resultChan <- matches
		}
	}
}

func (s *BruteForceScanner) findAllInMap(ctx context.Context, p *Process, patterns []Pattern, m maps.Map, buffer []byte) ([]Match, error) {
	var matches []Match
	maxPatternLen := 0
	for _, p := range patterns {
		if len(p.Value) > maxPatternLen {
			maxPatternLen = len(p.Value)
		}
	}

	overlap := make([]byte, 0, maxPatternLen-1)

	currentAddr := m.Start()
	memReader := NewMemReader(p.PID, WithFilter(func(m maps.Map) bool {
		return m.PathName() == "[heap]" || m.PathName() == ""
	}))

	for currentAddr < m.End() {

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, err := memReader.Seek(int64(currentAddr), io.SeekStart)
		if err != nil {
			var readErr ReadError
			if errors.As(err, &readErr) {
				currentAddr = readErr.nextValid
				continue
			}
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
					matches = append(matches, Match{Address: matchAddr, PatternIndex: i, Map: m})
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
