package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/vitaminmoo/memtools/maps"
)

// SimpleScanner implements the Scanner interface with a basic, non-parallel approach.
type SimpleScanner struct{}

func (s *SimpleScanner) Find(ctx context.Context, p *Process, patterns []Pattern) ([]Match, error) {
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

	memReader := NewMemReader(p.PID)

	for _, m := range allMaps {
		if !m.PermRead() {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, err := memReader.Seek(int64(m.Start()), io.SeekStart)
		if err != nil {
			// Could be a transient map, just skip it.
			continue
		}

		mapData := make([]byte, m.End()-m.Start())
		_, err = io.ReadFull(memReader, mapData)
		if err != nil {
			// Can't read the full map, maybe it changed. Skip.
			continue
		}

		for i, pattern := range patterns {
			searchLen := len(pattern.Value)
			// This is a naive implementation that will re-scan for each pattern.
			for j := 0; j <= len(mapData)-searchLen; {
				idx := bytes.Index(mapData[j:], pattern.Value)
				if idx != -1 {
					matchAddr := int64(m.Start()) + int64(j) + int64(idx)
					allMatches = append(allMatches, Match{Address: matchAddr, PatternIndex: i, Map: m})
					j += idx + 1
				} else {
					break // No more matches for this pattern in this map
				}
			}
		}
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Address < allMatches[j].Address
	})

	return allMatches, nil
}
