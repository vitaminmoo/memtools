//go:generate go run ../../../cmd/hexpatgen -i elf.hexpat -o elf_gen.go -pkg main

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vitaminmoo/memtools/hexpat/runtime"
	"github.com/vitaminmoo/memtools/maps"
	"github.com/vitaminmoo/memtools/process"
)

func main() {
	pid := os.Getpid()
	proc := process.New(pid)
	fmt.Printf("Reading own process: PID %d\n\n", pid)

	// Read memory maps and find the executable's base address.
	regions, err := maps.Read(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading maps: %v\n", err)
		os.Exit(1)
	}

	var base uintptr
	var exePath string
	for _, m := range regions {
		if m.Offset() == 0 && m.PermRead() && m.PermExecute() && m.PathName() != "" {
			base = m.Start()
			exePath = m.PathName()
			break
		}
	}
	if base == 0 {
		fmt.Fprintln(os.Stderr, "could not find executable mapping")
		os.Exit(1)
	}
	fmt.Printf("Executable: %s\n", exePath)
	fmt.Printf("Base addr:  0x%X\n\n", base)

	// Print a few interesting memory regions.
	fmt.Println("Memory regions:")
	shown := 0
	for _, m := range regions {
		if m.PathName() != "" && shown < 8 {
			fmt.Printf("  %s\n", m)
			shown++
		}
	}
	fmt.Println()

	// Use the Process (io.ReadSeeker) to read the ELF header from live memory.
	ctx := runtime.NewReadContext(proc)

	ident, errs := ReadELF(ctx, base)
	if errs.HasFatal() {
		fmt.Fprintf(os.Stderr, "read errors: %v\n", errs)
		os.Exit(1)
	}

	// E_IDENT is 16 bytes in the binary (4 magic + 5 fields + 7 padding).
	ehdr, errs := ReadElf64Ehdr(ctx, base+16)
	if errs.HasFatal() {
		fmt.Fprintf(os.Stderr, "read errors: %v\n", errs)
		os.Exit(1)
	}

	fmt.Printf("ELF Header (read from process memory at 0x%X):\n\n", base)

	out, _ := json.MarshalIndent(struct {
		Ident    *ELF
		Elf64Hdr *Elf64Ehdr
	}{ident, ehdr}, "", "  ")
	fmt.Println(string(out))
}
