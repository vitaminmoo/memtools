// Package process provides functionality to read memory from a process.
package process

import (
	"context"
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

	memReader := NewMemReader(p.PID)
	_, err := memReader.Seek(int64(m.Start()), io.SeekStart)
	if err != nil {
		return nil, nil // Map disappeared, fine to ignore.
	}

	mapData := make([]byte, m.End()-m.Start())
	n, err := io.ReadFull(memReader, mapData)
	if err != nil {
		return nil, nil // Map changed, fine to ignore.
	}
	mapData = mapData[:n]

	// The optimized search logic
	for i, b := range mapData {
		if anchorInfos, ok := anchors[b]; ok {
			for _, info := range anchorInfos {
				// Potential match found, check if the full pattern fits
				patternStartIndex := i - info.anchorOffset
				patternEndIndex := patternStartIndex + info.patternLen
				if patternStartIndex >= 0 && patternEndIndex <= len(mapData) {
					// Perform full masked comparison
					if findMasked(mapData[patternStartIndex:patternEndIndex], patterns[info.patternIndex]) == 0 {
						matchAddr := int64(m.Start()) + int64(patternStartIndex)
						matches = append(matches, Match{Address: matchAddr, PatternIndex: info.patternIndex, Map: m})
					}
				}
			}
		}
	}

	return matches, nil
}
