# ImHex Pattern Language Reference

A C-like data description language for parsing and annotating binary data in the ImHex hex editor. Files use the `.hexpat` extension.

## Primitive Types

| Unsigned | Signed | Other |
|----------|--------|-------|
| `u8` `u16` `u24` `u32` `u48` `u64` `u96` `u128` | `s8` `s16` `s24` `s32` `s48` `s64` `s96` `s128` | `float` `double` `bool` `char` `char16` `str` `padding` `auto` |

## Endianness

Prefix any type or type definition with `le` (little-endian) or `be` (big-endian):
```c
be u32 value @ 0x00;
le struct MyStruct { ... };
```
Default endianness is set via `#pragma endian big|little|native`.

## Pattern Placement (`@`)

Variables are overlaid onto binary data at a given offset:
```c
u32 magic @ 0x00;           // At absolute address
u16 field @ $;               // At current cursor position
MyStruct header @ 0x10;
```

## Dollar Operator (`$`)

`$` is the current read cursor position. It advances as patterns are placed. It can be read and assigned:
```c
u32 pos = $;
$ += 0x10;                   // Skip 16 bytes
```

## Structs

```c
struct Header {
    u32 magic;
    u16 version;
    u8 flags;
    padding[5];              // 5 bytes of padding
};
Header hdr @ 0x00;
```

Members are laid out sequentially from the placement address. Use `parent` to reference the enclosing struct and `this` to reference the current one.

## Unions

All members share the same starting offset. Size equals the largest member:
```c
union Value {
    u32 as_int;
    float as_float;
    u8 as_bytes[4];
};
```

## Enums

```c
enum FileType : u16 {
    PNG  = 0x5089,
    JPEG = 0xD8FF,
    BMP  = 0x4D42,
    Unknown = 0x100 ... 0x1FF  // Value range
};
```

## Bitfields

Define fields at the bit level:
```c
bitfield Flags {
    has_alpha : 1;
    compression : 3;
    reserved : 4;
};
```

Use `[[bitfield_order(direction, size)]]` to control bit ordering. Can use `be`/`le` prefix.

## Arrays

```c
u8 data[16];                          // Fixed-size array
u8 name[while(std::mem::read_unsigned($, 1) != 0x00)]; // Condition-based
char str[];                           // Null-terminated string shorthand
u32 items[parent.count];              // Size from another field
```

## Pointers

```c
u32 *ptr : u32 @ 0x10;               // Pointer: dereferences a u32 at the address read from offset 0x10
MyStruct *entries[count] : u16 @ 0x20; // Array of pointers
```

Use `[[pointer_base("func_name")]]` to customize address calculation.

## Parameterized Types (Templates)

```c
struct Array<T, auto Count> {
    T entries[Count];
};
Array<u32, 10> my_array @ 0x00;
```

## Struct Inheritance

```c
struct Base {
    u32 id;
};
struct Derived : Base {
    u16 extra;              // Follows Base's fields in memory
};
```

## Type Aliases

```c
using DWORD = u32;
using Vec3<T> = struct { T x, y, z; };
```

## Enums as Struct Members with Match

```c
struct Packet {
    u8 type;
    match (type) {
        (0x01): u32 int_payload;
        (0x02): float float_payload;
        (_):    padding[4];           // _ is wildcard
    }
};
```

Match can take multiple values: `match(a, b) { (1, 2): ...; }`.

## Functions

```c
fn sum(u32 a, u32 b) {
    return a + b;
};

fn format_size(auto value) {
    return std::format("{} bytes", value);
};
```

## Control Flow

```c
if (header.version >= 2) {
    u32 extra_field;
} else {
    padding[4];
}

u32 count;
u32 total = 0;
for (u32 i = 0, i < count, i = i + 1) {   // Note: commas, not semicolons
    total = total + 1;
}

while ($ < end_offset) {
    Entry entry;
}

break;      // Exit loop
continue;   // Next iteration
```

## Try-Catch

```c
try {
    RiskyStruct data;
} catch {
    padding[sizeof(RiskyStruct)];
}
```

## Namespaces

```c
namespace Format {
    struct Header { u32 magic; };
    fn validate() { return true; };
};
Format::Header hdr @ 0x00;
```

## Operators

| Category | Operators |
|----------|-----------|
| Arithmetic | `+` `-` `*` `/` `%` |
| Bitwise | `&` `\|` `^` `~` `<<` `>>` |
| Comparison | `==` `!=` `<` `>` `<=` `>=` |
| Logical | `&&` `\|\|` `^^` `!` |
| Assignment | `=` |
| Ternary | `condition ? a : b` |
| Type | `addressof(x)` `sizeof(x)` `typenameof(x)` |

## Attributes

Applied with `[[attr]]` or `[[attr(args)]]` after a type definition or variable:

| Attribute | Description |
|-----------|-------------|
| `[[color("RRGGBB")]]` | Set highlight color |
| `[[name("label")]]` | Custom display name |
| `[[comment("text")]]` | Add tooltip comment |
| `[[format("func")]]` | Custom display format function |
| `[[format_read("func")]]` | Custom read format function |
| `[[format_write("func")]]` | Custom write format function |
| `[[format_entries("func")]]` | Format each array entry |
| `[[transform("func")]]` | Transform value after reading |
| `[[transform_entries("func")]]` | Transform each array entry |
| `[[pointer_base("func")]]` | Custom pointer base address |
| `[[fixed_size(N)]]` | Pad type to N bytes |
| `[[hidden]]` | Hide from pattern data view |
| `[[highlight_hidden]]` | Hide hex highlighting only |
| `[[inline]]` | Inline nested members into parent |
| `[[merge]]` | Merge members into parent scope |
| `[[sealed]]` | Prevent modification |
| `[[static]]` | Mark array as static type |
| `[[single_color]]` | Use one color for all members |
| `[[no_unique_address]]` | Allow overlapping addresses |
| `[[export]]` | Export to parent scope |
| `[[bitfield_order(dir, size)]]` | Control bitfield bit ordering |

## Preprocessor

```c
#include "other.hexpat"
#pragma once

#define MAGIC 0x464C457F
#ifdef MAGIC
    // ...
#endif
#undef MAGIC
#error "message"
```

## Pragmas

```c
#pragma endian big|little|native    // Default endianness
#pragma eval_depth 100              // Recursion limit (0 = unlimited)
#pragma array_limit 0x1000          // Max array elements (0 = unlimited)
#pragma pattern_limit 0x2000        // Max patterns (0 = unlimited)
#pragma loop_limit 0x1000           // Max loop iterations (0 = unlimited)
#pragma debug                       // Enable debug output
#pragma allow_edits                 // Allow writing to data
```

## Literals

```c
42          // Decimal
0xFF        // Hexadecimal
0b10101010  // Binary
0o77        // Octal
3.14        // Float
3.14F       // Explicit float
1e-5        // Scientific
'A'         // Character
'\x41'      // Hex character escape
'\u0041'    // Unicode escape
"string"    // String
true false  // Boolean
null        // Null
```

## Standard Library (`std::`)

### Core I/O
```c
std::print("value = {}", value);     // Print to console
std::format("0x{:02X}", byte);       // Format string (returns str)
std::error("message");               // Abort with error
std::warning("message");             // Log warning
std::assert(cond, "message");        // Assert condition
std::env("VAR_NAME");                // Read environment variable
```

### Memory (`std::mem::`)
```c
std::mem::base_address()                                       // Start address
std::mem::size()                                               // Data size
std::mem::read_unsigned(addr, byte_count, endian)              // Read uint
std::mem::read_signed(addr, byte_count, endian)                // Read sint
std::mem::read_string(addr, length)                            // Read string
std::mem::find_sequence_in_range(n, start, end, bytes...)      // Find nth byte sequence
std::mem::find_string_in_range(n, start, end, string)          // Find nth string
std::mem::current_bit_offset()                                 // Bit offset in bitfields
std::mem::read_bits(byte_off, bit_off, bit_size)               // Read bits

// Sections (virtual memory regions)
std::mem::create_section("name")         // Create, returns section ID
std::mem::delete_section(id)
std::mem::get_section_size(id)
std::mem::set_section_size(id, size)
std::mem::copy_to_section(from_id, from_addr, to_id, to_addr, size)
std::mem::copy_value_to_section(value, section_id, addr)
```

### String (`std::string::`)
```c
std::string::length(s)
std::string::at(s, index)
std::string::substr(s, pos, count)
std::string::parse_int(s, base)
std::string::parse_float(s)
```

### Math (`std::math::`)
```c
std::math::floor(x)  ceil(x)  round(x)  trunc(x)
std::math::log2(x)  log10(x)  ln(x)
std::math::pow(x,y)  exp(x)  sqrt(x)  cbrt(x)  fmod(x,y)
std::math::sin(x)  cos(x)  tan(x)  asin(x)  acos(x)  atan(x)  atan2(y,x)
std::math::sinh(x)  cosh(x)  tanh(x)  asinh(x)  acosh(x)  atanh(x)
std::math::accumulate(start, end, size, section, operation, endian)
```

### Core Reflection (`std::core::`)
```c
std::core::array_index()                           // Current index in array evaluation
std::core::member_count(pattern)                    // Number of members
std::core::has_member(pattern, "name")              // Check member exists
std::core::formatted_value(pattern)                 // Get formatted display string
std::core::is_valid_enum(pattern)                   // Check enum value is defined
std::core::set_display_name(pattern, "name")        // Override display name
std::core::set_pattern_color(pattern, 0xRRGGBB)     // Override color
std::core::set_pattern_comment(pattern, "text")     // Set comment
std::core::set_endian(endian)                       // Change endianness at runtime
std::core::get_endian()                             // Get current endianness
std::core::has_attribute(pattern, "attr_name")       // Check for attribute
std::core::execute_function("name", args...)         // Call function by name
```

### Hashing (`std::hash::`)
```c
std::hash::crc32(pattern, init, poly, xorout, reflect_in, reflect_out)
// Also: crc8, crc16, crc64 with same signature
```

### Time (`std::time::`)
```c
std::time::epoch()                     // Current unix timestamp
std::time::to_local(timestamp)         // To local time struct
std::time::to_utc(timestamp)           // To UTC time struct
std::time::format("fmt", time_struct)  // Format time string
```

### File I/O (`std::file::`) — requires permission
```c
std::file::open(path, mode)    // mode: 1=Read, 2=Write, 3=Create
std::file::close(handle)
std::file::read(handle, size)
std::file::write(handle, data)
std::file::seek(handle, offset)
std::file::size(handle)
```

## Endianness Constants

`std::core::set_endian()` and related functions use: `0` = Native, `1` = Big, `2` = Little.

## Doc Comments

```c
/*!
 * Global pattern description shown in ImHex
 */

/// Field-level doc comment
u32 magic;
```

## Complete Example

```c
#pragma endian little
#pragma pattern_limit 0x10000

import std.mem;

struct String {
    u32 length;
    char data[length];
} [[format("format_string"), color("00FF00")]];

fn format_string(String s) {
    return s.data;
};

enum ChunkType : u32 {
    Header = 0x01,
    Data   = 0x02,
    End    = 0xFF
};

struct Chunk {
    ChunkType type;
    u32 size;
    match (type) {
        (ChunkType::Header): u8 header_data[size];
        (ChunkType::Data):   u8 payload[size];
        (_):                 padding[size];
    }
} [[color("FF8800")]];

struct File {
    u32 magic;
    std::assert(magic == 0x46494C45, "Bad magic");
    u16 chunk_count;
    Chunk chunks[chunk_count];
};

File file @ 0x00;
```
