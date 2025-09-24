// Package testutil provides utilities for setting up and tearing down
// a benchmark environment for memory scanning tests.
package testutil

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
	BenchmarkPID      int
	BenchmarkPatterns [][]byte
	BenchmarkMatches  []uintptr
	BenchmarkBaseAddr int64
	BenchmarkMemSize  int64
)

const memSizeMB = 2048

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
	BenchmarkMemSize = memSizeMB * 1024 * 1024
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
	BenchmarkPID = targetCmd.Process.Pid

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
			BenchmarkBaseAddr = addr
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

			BenchmarkPatterns = append(BenchmarkPatterns, patternVal)
			BenchmarkMatches = append(BenchmarkMatches, uintptr(addr))
		} else if line == "READY" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading benchmark target output: %v", err)
		os.Exit(1)
	}

	// Ensure we have what we need
	if BenchmarkBaseAddr == 0 || len(BenchmarkPatterns) != 3 || len(BenchmarkMatches) != 3 {
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
