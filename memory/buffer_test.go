package memory

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/testutil"
)

func TestMain(m *testing.M) {
	testutil.TestMain(m)
}

func TestReadKnownValues(t *testing.T) {
	pid := testutil.BenchmarkPID
	procMaps, err := maps.Read(pid)
	require.NoError(t, err)
	for i, pattern := range testutil.BenchmarkPatterns {
		match := testutil.BenchmarkMatches[i]
		targetMap, err := procMaps.Find(match)
		require.NoError(t, err)
		buffer := NewBuffer(pid, match, targetMap.End(), len(pattern))
		result, err := buffer.Next(len(pattern))
		require.NoError(t, err, "Failed to read memory at address 0x%x", match)
		require.Equal(t, pattern, result, "Memory content mismatch at address 0x%x", match)
		t.Logf("PATTERN_%d: %x @ %x", i+1, result, match)
	}
}

func TestMemBuffer(t *testing.T) {
	now := time.Now()
	pid := testutil.BenchmarkPID
	procMaps, err := maps.Read(pid)
	require.NoError(t, err)
	pattern := testutil.BenchmarkPatterns[0]
	match := testutil.BenchmarkMatches[0]
	targetMap, err := procMaps.Find(match)
	require.NoError(t, err)

	buf := NewBuffer(pid, match, targetMap.End(), 0)
	err = buf.Refill()
	require.NoError(t, err)

	found, err := buf.Next(32)
	require.NoError(t, err)
	require.Equal(t, pattern, found)
	read := len(found)
	for {
		found, err := buf.Next(1024 * 1024)
		if err != nil && err == io.EOF {
			fmt.Printf("Reached EOF after reading %dMB\n", read/(1024*1024))
			fmt.Printf("Total time: %s\n", time.Since(now))
			fmt.Printf("Read rate: %.2f MB/s\n", float64(read)/(1024*1024)/time.Since(now).Seconds())
			break
		}
		require.NoError(t, err)
		require.NotZero(t, len(found))
		read += len(found)
	}
}

func BenchmarkMemBuffer(b *testing.B) {
	b.ReportAllocs()
	pid := testutil.BenchmarkPID
	procMaps, err := maps.Read(pid)
	require.NoError(b, err)
	// pattern := testutil.BenchmarkPatterns[0]
	match := testutil.BenchmarkMatches[0]
	targetMap, err := procMaps.Find(match)
	require.NoError(b, err)

	for _, size := range []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192} {
		b.Run(fmt.Sprintf("%dB reads small buffer", size), func(b *testing.B) {
			do(b, pid, targetMap.Start(), targetMap.End(), 1024*512, size)
		})
	}

	for _, size := range []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192} {
		b.Run(fmt.Sprintf("%dB reads big buffer", size), func(b *testing.B) {
			do(b, pid, targetMap.Start(), targetMap.End(), 1024*1024, size)
		})
	}
}

func do(b *testing.B, pid int, start, end uintptr, bufLen, readLen int) {
	buf := NewBuffer(pid, start, end, bufLen)
	// buf.Reset(start)
	for b.Loop() {
		_, err := buf.Next(readLen)
		if err != nil {
			if err == io.EOF {
				buf.Reset(start)
				continue
			}
			b.Errorf("unexpected error: %v", err)
		}
	}
	b.ReportMetric(float64(buf.syscalls)/float64(b.N), "syscalls/op")
	b.ReportMetric(float64(b.Elapsed())/float64(b.N)/float64(readLen), "ns/byte")
}
