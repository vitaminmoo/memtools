// Package process provides functionality to read memory from a process.
package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sort"
	"sync"

	"github.com/vitaminmoo/memtools/maps"
)

// OptimizedScanner implements the Scanner interface using a parallelized
// search with a pre-computed anchor-byte lookup table to screen for potential matches.
type OptimizedScanner struct{}

// anchorTable maps a byte value to a list of pattern indices that use it as an anchor.
type anchorTable map[byte][]*anchorInfo

// anchorInfo contains the pattern index and its length.
type anchorInfo struct {
	patternIndex int
	patternLen   int
	anchorOffset int
}

func (s *OptimizedScanner) Find(ctx context.Context, p *Process, patterns []Pattern) ([]Match, error) {
	for i, pattern := range patterns {
		if len(pattern.Value) == 0 {
			return nil, fmt.Errorf("pattern %d is empty", i)
		}
		if len(pattern.Mask) > 0 && len(pattern.Value) != len(pattern.Mask) {
			return nil, fmt.Errorf("value and mask length mismatch for pattern %d", i)
		}
	}

	anchors, maxPatternLen := s.buildAnchorTable(patterns)

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
			s.findAllWorker(ctx, p, patterns, anchors, maxPatternLen, readableMaps, resultChan)
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

func (s *OptimizedScanner) buildAnchorTable(patterns []Pattern) (anchorTable, int) {
	table := make(anchorTable)
	maxPatternLen := 0
	for i, p := range patterns {
		if len(p.Value) > maxPatternLen {
			maxPatternLen = len(p.Value)
		}

		anchorOffset := -1
		// Prefer the first non-masked byte as an anchor.
		if len(p.Mask) > 0 {
			for j, maskByte := range p.Mask {
				if maskByte == 0xFF {
					anchorOffset = j
					break
				}
			}
		} else { // No mask, first byte is the anchor.
			anchorOffset = 0
		}

		// If no perfect anchor, use the first byte.
		if anchorOffset == -1 && len(p.Value) > 0 {
			anchorOffset = 0
		}

		if anchorOffset != -1 {
			anchorByte := p.Value[anchorOffset]
			info := &anchorInfo{
				patternIndex: i,
				patternLen:   len(p.Value),
				anchorOffset: anchorOffset,
			}
			table[anchorByte] = append(table[anchorByte], info)
		}
	}
	return table, maxPatternLen
}

func (s *OptimizedScanner) findAllWorker(ctx context.Context, p *Process, patterns []Pattern, anchors anchorTable, maxPatternLen int, mapsChan <-chan maps.Map, resultChan chan<- []Match) {
	buffer := make([]byte, findChunkSize+maxPatternLen-1)

	for m := range mapsChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		matches, err := s.findAllInMap(ctx, p, patterns, anchors, m, buffer)
		if err == nil && len(matches) > 0 {
			resultChan <- matches
		}
	}
}

func (s *OptimizedScanner) findAllInMap(ctx context.Context, p *Process, patterns []Pattern, anchors anchorTable, m maps.Map, buffer []byte) ([]Match, error) {
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

		// The optimized search logic
		for i, b := range dataToSearch {
			if anchorInfos, ok := anchors[b]; ok {
				for _, info := range anchorInfos {
					// Potential match found, check if the full pattern fits
					patternStartIndex := i - info.anchorOffset
					patternEndIndex := patternStartIndex + info.patternLen
					if patternStartIndex >= 0 && patternEndIndex <= len(dataToSearch) {
						// Perform full masked comparison
						if findMasked(dataToSearch[patternStartIndex:patternEndIndex], patterns[info.patternIndex]) == 0 {
							matchAddr := readStartAddr + int64(patternStartIndex) - int64(len(overlap))
							matches = append(matches, Match{Address: matchAddr, PatternIndex: info.patternIndex, Map: m})
						}
					}
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
