package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sort"
	"sync"

	"github.com/vitaminmoo/memtools/maps"
)

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
