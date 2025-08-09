package process

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var (
	benchmarkPID      int
	benchmarkPatterns []Pattern
	benchmarkMatches  []Match
	benchmarkBaseAddr int64
	benchmarkMemSize  int64
)

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Setting up benchmark environment...")

	// Compile the benchmark target
	compileCmd := exec.Command("make", "-C", "../benchmark_target")
	output, err := compileCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to compile benchmark target: %v\n%s", err, output)
		os.Exit(1)
	}

	// Run the benchmark target
	const memSizeMB = 1024
	benchmarkMemSize = memSizeMB * 1024 * 1024
	targetCmd := exec.Command("../benchmark_target/benchmark_target")
	targetCmd.Env = append(os.Environ(), fmt.Sprintf("ALLOCATE_MEM_MB=%d", memSizeMB))
	stdout, err := targetCmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to get stdout pipe: %v", err)
		os.Exit(1)
	}
	stdin, err := targetCmd.StdinPipe()
	if err != nil {
		fmt.Printf("Failed to get stdin pipe: %v", err)
		os.Exit(1)
	}

	if err := targetCmd.Start(); err != nil {
		fmt.Printf("Failed to start benchmark target: %v", err)
		os.Exit(1)
	}

	fmt.Printf("Benchmark target started with PID: %d\n", targetCmd.Process.Pid)
	benchmarkPID = targetCmd.Process.Pid

	// Parse the output from the C++ program
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		if strings.HasPrefix(line, "PID:") {
			// We already have the PID from targetCmd.Process.Pid
		} else if strings.HasPrefix(line, "BASE_ADDRESS:") {
			addrStr := strings.TrimSpace(strings.Split(line, ":")[1])
			addr, err := strconv.ParseInt(addrStr, 16, 64)
			if err != nil {
				fmt.Printf("Failed to parse base address hex: %v", err)
				os.Exit(1)
			}
			benchmarkBaseAddr = addr
		} else if strings.HasPrefix(line, "PATTERN_") {
			parts := strings.Split(line, " ")
			patternStr := parts[1]
			addrStr := parts[3]

			patternVal, err := hex.DecodeString(patternStr)
			if err != nil {
				fmt.Printf("Failed to decode pattern hex: %v", err)
				os.Exit(1)
			}
			addr, err := strconv.ParseInt(addrStr, 16, 64)
			if err != nil {
				fmt.Printf("Failed to parse address hex: %v", err)
				os.Exit(1)
			}

			benchmarkPatterns = append(benchmarkPatterns, Pattern{Value: patternVal})
			benchmarkMatches = append(benchmarkMatches, Match{Address: addr, PatternIndex: len(benchmarkPatterns) - 1})
		} else if line == "READY" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading benchmark target output: %v", err)
		os.Exit(1)
	}

	// Ensure we have what we need
	if benchmarkBaseAddr == 0 || len(benchmarkPatterns) != 3 || len(benchmarkMatches) != 3 {
		fmt.Println("Failed to parse all required info from benchmark target")
		os.Exit(1)
	}

	// Run tests
	exitCode := m.Run()

	// Teardown
	fmt.Println("Tearing down benchmark environment...")
	fmt.Fprintln(stdin, "")
	targetCmd.Wait()
	fmt.Println("Benchmark target exited.")

	os.Exit(exitCode)
}

func verifyAndLogResults(b *testing.B, foundMatches []Match, expectedMatches []Match) {
	b.StopTimer()

	foundMap := make(map[int64]bool)
	for _, found := range foundMatches {
		if found.Address >= benchmarkBaseAddr && found.Address < (benchmarkBaseAddr+benchmarkMemSize) {
			foundMap[found.Address] = true
		}
	}

	verifiedCount := 0
	for _, expected := range expectedMatches {
		if foundMap[expected.Address] {
			verifiedCount++
		}
	}

	b.Logf("Found %d matches in the target memory block.", verifiedCount)

	if verifiedCount != len(expectedMatches) {
		b.Errorf("Expected to find %d matches, but only found %d in the target block", len(expectedMatches), verifiedCount)
	}

	scanSizeMB := float64(1024) // Since we allocated 1024MB
	throughput := scanSizeMB / b.Elapsed().Seconds()
	b.Logf("Total scanned: %.2f MB. Throughput: %.2f MB/s", scanSizeMB, throughput)
}

// --- Benchmarks ---

func BenchmarkBruteForceScanner_Find_MultiPattern(b *testing.B) {
	p := New(benchmarkPID)
	p.Scanner = &BruteForceScanner{}

	b.ResetTimer()
	var found []Match
	var err error
	for b.Loop() {
		found, err = p.Find(b.Context(), benchmarkPatterns)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
	}
	verifyAndLogResults(b, found, benchmarkMatches)
}

func BenchmarkOptimizedScanner_Find_MultiPattern(b *testing.B) {
	p := New(benchmarkPID)
	p.Scanner = &OptimizedScanner{}

	b.ResetTimer()
	var found []Match
	var err error
	for b.Loop() {
		found, err = p.Find(b.Context(), benchmarkPatterns)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
	}
	verifyAndLogResults(b, found, benchmarkMatches)
}
