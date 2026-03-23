# hexpatgen

CLI tool that compiles [ImHex pattern language](https://docs.werwolv.net/pattern-language/) (`.hexpat`) files into Go structs and typed reader functions. The generated code reads from any `io.ReadSeeker` — process memory, files, byte buffers, etc.

## Install

```sh
go install github.com/vitaminmoo/memtools/cmd/hexpatgen@latest
```

## Usage

```sh
# Generate Go source to stdout
hexpatgen -i structs.hexpat

# Generate to a file with a specific package name
hexpatgen -i structs.hexpat -o structs_gen.go -pkg mypackage
```

## Generated Code

For each struct in the `.hexpat` file, the generator emits:

- A plain Go struct with exported fields
- A `Read<StructName>(ctx *runtime.ReadContext, addr uintptr) (*StructName, runtime.Errors)` function that eagerly reads all fields and collects errors without aborting

Errors carry full context via `runtime.ChainError` (field path + address + underlying error).

## Supported Features

- Structs with implicit and explicit (`@ address`) field offsets
- Per-field and global (`#pragma endian`) endianness
- All primitive types: `u8`-`u128`, `s8`-`s128`, `float`, `double`, `char`, `char16`, `bool`
- Enums with underlying type and auto-increment
- Unions and bitfields
- Fixed-size and expression-sized arrays
- Pointer types (`u16 *ptr : u32`)
- Conditional fields (`if`/`else`)
- Struct inheritance
- `using` type aliases
- `padding[N]`

## Examples

- **[`example/`](example/)** — Parse ELF headers from the running process's own memory using generated code from a full ELF `.hexpat` pattern.

## Building

```sh
go build ./cmd/hexpatgen
```
