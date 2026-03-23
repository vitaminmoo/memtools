# memtools

Go toolkit for reading and analyzing process memory structures on Linux.

## Packages

### Core Memory Access

These packages provide `io.ReadSeeker` access to live process memory, which can be fed directly into `hexpatgen`-generated readers or used standalone.

- **`process/`** — Read process memory via `ptrace` / `ProcessVMReadv`. Find processes by PID or name, read arbitrary addresses through an `io.ReadSeeker` interface.
- **`memory/`** — Buffered memory reader with configurable buffer size (default 1MB). Wraps raw syscalls for efficient sequential reads.
- **`maps/`** — Parse `/proc/[pid]/maps` to query memory regions, permissions, and mapped files.

### ImHex Pattern Language

- **[`cmd/hexpatgen/`](cmd/hexpatgen/README.md)** — CLI tool that compiles [ImHex pattern language](https://docs.werwolv.net/pattern-language/) `.hexpat` files into Go structs and typed reader functions. Works with any `io.ReadSeeker` (process memory, files, byte buffers). No reflection overhead.
- **`hexpat/parser/`** — Parser for the ImHex pattern language. Produces a full AST from `.hexpat` source. 96% compatible with upstream ImHex-Patterns.
- **`hexpat/resolve/`** — Type resolution from parser AST to intermediate representation
- **`hexpat/codegen/`** — Go source emission from resolved IR
- **`hexpat/runtime/`** — Helpers used by generated code (read context, cycle detection, structured errors)

### Reflection-based (deprecated)

- **`sparsestruct/`** — Reads binary structures using Go reflection and struct tags (`offset:"0x100,be"`). Supports pointer chasing, cycle detection, and C struct export for Ghidra. Being replaced by `hexpatgen` which is significantly faster.

### Reverse Engineering Integration

- **`ghidra_scripts/`** — Python scripts for [Ghidra](https://ghidra-sre.org/) that recursively apply type information to struct pointer fields. Pair with `sparsestruct.GenerateCDefinitions()` or hand-written C headers.

## Quick Start

```sh
# Install the code generator
go install github.com/vitaminmoo/memtools/cmd/hexpatgen@latest

# Generate Go code from a pattern file
hexpatgen -i structs.hexpat -o structs_gen.go -pkg mypackage

# Use in your code
go build ./...
```

## Requirements

- Go 1.24+
- Linux (uses `ProcessVMReadv` and `/proc` filesystem)
