package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

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

	var allMatches []Match
	maxPatternLen := 0
	for _, p := range patterns {
		if len(p.Value) > maxPatternLen {
			maxPatternLen = len(p.Value)
		}
	}
	buffer := make([]byte, findChunkSize+maxPatternLen-1)

	for _, m := range allMaps {
		if !m.PermRead() {
			continue
		}
		select {
		case <-ctx.Done():
			break
		default:
		}
		matches, err := s.findAllInMap(ctx, p, patterns, m, buffer)
		if err == nil && len(matches) > 0 {
			allMatches = append(allMatches, matches...)
		}
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Address < allMatches[j].Address
	})

	return allMatches, nil
}

func (s *BruteForceScanner) findAllInMap(ctx context.Context, p *Process, patterns []Pattern, m maps.Map, buffer []byte) ([]Match, error) {
	var matches []Match

	memReader := NewMemReader(p.PID)
	_, err := memReader.Seek(int64(m.Start()), io.SeekStart)
	if err != nil {
		// Map may have disappeared, which is fine.
		return nil, nil
	}

	mapData := make([]byte, m.End()-m.Start())
	n, err := io.ReadFull(memReader, mapData)
	if err != nil {
		// Map changed during read, also fine.
		return nil, nil
	}
	mapData = mapData[:n]

	for i, pattern := range patterns {
		searchLen := len(pattern.Value)
		for j := 0; j <= len(mapData)-searchLen; {
			// Use findMasked which handles both masked and unmasked patterns
			data := mapData[j:]
			offset := findMasked(data, pattern)
			if offset != -1 {
				matchAddr := int64(m.Start()) + int64(j) + int64(offset)
				matches = append(matches, Match{Address: matchAddr, PatternIndex: i, Map: m})
				j += offset + 1
			} else {
				break // No more matches for this pattern in this map
			}
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
