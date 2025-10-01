package memory

import (
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
)

type matchState struct {
	matchAny bool
	b        byte
}

func compile(pattern string) []matchState {
	size := len(pattern) / 2
	program := make([]matchState, size)
	for i := range size {
		bs := pattern[:2]
		pattern = pattern[2:]
		if bs == "??" {
			program[i] = matchState{matchAny: true}
		} else {
			b, err := hex.DecodeString(bs)
			if err != nil {
				panic("invalid hex in pattern")
			}
			program[i] = matchState{b: b[0]}
		}
	}
	return program
}

func (s *matchState) matches(b byte) bool {
	return s.matchAny || s.b == b
}

func (b *Buffer) FindFirst(pattern string) (uintptr, error) {
	b.Reset(b.start)
	return b.Find(pattern)
}

func (b *Buffer) Find(pattern string) (uintptr, error) {
	if len(pattern) == 0 {
		return 0, nil
	}
	pattern = stripWhitespace(pattern)
	if len(pattern)%2 != 0 {
		return 0, errors.New("pattern length must be even")
	}

	program := compile(pattern)
	patternLen := len(program)

	var absolutePos int

BUF:
	for {
		data, err := b.Next(1024 * 1024)
		if err != nil {
			return 0, err
		}

		// Iterate through each possible starting position in the data.
		for i := range len(data) - patternLen - 1 {
			foundMatch := true
			// Attempt to match the entire pattern from this starting position.
			for j := range patternLen {
				// If any part of the pattern doesn't match, this is not a valid start.
				if !program[j].matches(data[i+j]) {
					foundMatch = false
					break
				}
			}
			// If the inner loop completed without a mismatch, we found it!
			if foundMatch {
				return uintptr(b.start + uintptr(absolutePos+i)), nil
			}
		}

		absolutePos += len(data)
		if patternLen > 1 {
			b.Rewind(patternLen - 1)
			absolutePos -= (patternLen - 1)
		}
		continue BUF
	}
}

func stripWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		if !unicode.IsSpace(c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}
