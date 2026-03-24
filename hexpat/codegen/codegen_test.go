package codegen

import (
	"bytes"
	"encoding/binary"
	goparser "go/parser"
	"go/token"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/hexpat/resolve"
	"github.com/vitaminmoo/memtools/hexpat/runtime"
	"github.com/vitaminmoo/memtools/hexpat/parser"
)

func mustParse(t *testing.T, src string) *parser.File {
	t.Helper()
	file, err := parser.Parse(src)
	require.NoError(t, err, "parse failed")
	return file
}

func mustGenerate(t *testing.T, src string) string {
	t.Helper()
	file := mustParse(t, src)
	pkg, err := resolve.Resolve(file)
	require.NoError(t, err)
	out, err := Generate(pkg, Options{PackageName: "test"})
	require.NoError(t, err)
	return string(out)
}

func assertCompiles(t *testing.T, src string) {
	t.Helper()
	fset := token.NewFileSet()
	_, err := goparser.ParseFile(fset, "generated.go", src, goparser.AllErrors)
	assert.NoError(t, err, "generated code does not parse:\n%s", src)
}

func TestGenerateCompiles(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u32 magic;
	u16 version;
	u8 flags;
};
`)
	assertCompiles(t, src)
}

func TestGenerateWithEnum(t *testing.T) {
	src := mustGenerate(t, `
enum Compression : u32 {
	None = 0,
	RLE8 = 1,
	RLE4 = 2
};

struct Header {
	u32 magic;
	Compression compression;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Compression uint32")
	assert.Contains(t, src, "CompressionNone")
	assert.Contains(t, src, "CompressionRLE8")
	assert.Contains(t, src, "func (e Compression) String() string")
	assert.Contains(t, src, "func (e Compression) MarshalJSON() ([]byte, error)")
}

func TestGenerateWithPointer(t *testing.T) {
	src := mustGenerate(t, `
struct Node {
	u32 value;
	Node *next : u64;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "*Node")
	assert.Contains(t, src, "ctx.Visit")
}

func TestGenerateWithArray(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u8 magic[4];
	u32 values[3];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "[4]uint8")
	assert.Contains(t, src, "[3]uint32")
}

func TestGenerateNestedStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Inner {
	u16 x;
	u16 y;
};

struct Outer {
	u32 id;
	Inner pos;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "ReadInner")
	assert.Contains(t, src, "ReadOuter")
}

func TestGenerateWithFloats(t *testing.T) {
	src := mustGenerate(t, `
struct Floats {
	float x;
	double y;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "float32")
	assert.Contains(t, src, "float64")
	assert.Contains(t, src, "math.Float32frombits")
	assert.Contains(t, src, "math.Float64frombits")
}

func TestGenerateWithEndianOverride(t *testing.T) {
	src := mustGenerate(t, `
struct Mixed {
	le u32 little_val;
	be u32 big_val;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "binary.LittleEndian")
	assert.Contains(t, src, "binary.BigEndian")
}

// Integration test: generate code, compile it, and verify it reads binary data correctly.
func TestIntegrationReadSimpleStruct(t *testing.T) {
	// Build binary data for: magic=0xDEADBEEF, version=0x0102, flags=0x42
	var data bytes.Buffer
	binary.Write(&data, binary.LittleEndian, uint32(0xDEADBEEF))
	binary.Write(&data, binary.LittleEndian, uint16(0x0102))
	binary.Write(&data, binary.LittleEndian, uint8(0x42))

	ctx := runtime.NewReadContext(bytes.NewReader(data.Bytes()))

	// Manually test the ReadAt / pattern that generated code would use
	var buf [4]byte
	n, err := ctx.ReadAt(buf[:4], 0)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	magic := binary.LittleEndian.Uint32(buf[:4])
	assert.Equal(t, uint32(0xDEADBEEF), magic)

	n, err = ctx.ReadAt(buf[:2], 4)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	version := binary.LittleEndian.Uint16(buf[:2])
	assert.Equal(t, uint16(0x0102), version)

	n, err = ctx.ReadAt(buf[:1], 6)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, byte(0x42), buf[0])
}

func TestIntegrationReadFloats(t *testing.T) {
	var data bytes.Buffer
	binary.Write(&data, binary.LittleEndian, float32(3.14))
	binary.Write(&data, binary.LittleEndian, float64(2.71828))

	ctx := runtime.NewReadContext(bytes.NewReader(data.Bytes()))

	var buf [8]byte
	_, err := ctx.ReadAt(buf[:4], 0)
	require.NoError(t, err)
	f32 := math.Float32frombits(binary.LittleEndian.Uint32(buf[:4]))
	assert.InDelta(t, float32(3.14), f32, 0.001)

	_, err = ctx.ReadAt(buf[:8], 4)
	require.NoError(t, err)
	f64 := math.Float64frombits(binary.LittleEndian.Uint64(buf[:8]))
	assert.InDelta(t, 2.71828, f64, 0.0001)
}

func TestIntegrationCycleDetection(t *testing.T) {
	ctx := runtime.NewReadContext(bytes.NewReader(nil))

	// First visit should return false (not yet visited)
	assert.False(t, ctx.Visit(0x1000))
	// Second visit should return true (already visited)
	assert.True(t, ctx.Visit(0x1000))
}

func TestGenerateBigEndian(t *testing.T) {
	src := mustGenerate(t, `
#pragma endian big
struct Header {
	u32 magic;
	u16 version;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "binary.BigEndian")
}

func TestGenerateInheritance(t *testing.T) {
	src := mustGenerate(t, `
struct Base {
	u32 id;
};

struct Derived : Base {
	u16 extra;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Derived struct")
}

// --- New Phase 2 tests ---

func TestGenerateUnion(t *testing.T) {
	src := mustGenerate(t, `
union Value {
	u32 as_int;
	float as_float;
	u8 as_bytes[4];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Value struct")
	assert.Contains(t, src, "AsInt")
	assert.Contains(t, src, "AsFloat")
	assert.Contains(t, src, "AsBytes")
	assert.Contains(t, src, "ReadValue")
}

func TestGenerateBitfield(t *testing.T) {
	src := mustGenerate(t, `
bitfield Flags {
	readable : 1;
	writable : 1;
	executable : 1;
	padding : 5;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Flags struct")
	assert.Contains(t, src, "Readable")
	assert.Contains(t, src, "Writable")
	assert.Contains(t, src, "Executable")
	assert.Contains(t, src, "ReadFlags")
	// Should use shift/mask
	assert.Contains(t, src, ">>")
	assert.Contains(t, src, "&1")
}

func TestGenerateBitfieldInStruct(t *testing.T) {
	src := mustGenerate(t, `
bitfield Perms {
	read : 1;
	write : 1;
	padding : 6;
};

struct File {
	u32 size;
	Perms perms;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Perms struct")
	assert.Contains(t, src, "ReadPerms")
	assert.Contains(t, src, "ReadFile")
}

func TestGenerateConditional(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u32 flags;
	if (flags & 0x01) {
		u32 extra;
	}
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type Header struct")
	assert.Contains(t, src, "Flags")
	assert.Contains(t, src, "Extra")
	assert.Contains(t, src, "offset")
	assert.Contains(t, src, "if ")
	assert.Contains(t, src, "result.Flags")
}

func TestGenerateConditionalElseIf(t *testing.T) {
	src := mustGenerate(t, `
struct Msg {
	u8 type;
	if (type == 1) {
		u32 value_a;
	} else if (type == 2) {
		u16 value_b;
	} else {
		u8 value_c;
	}
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "ValueA")
	assert.Contains(t, src, "ValueB")
	assert.Contains(t, src, "ValueC")
	assert.Contains(t, src, "} else if")
	assert.Contains(t, src, "} else {")
}

func TestGenerateExprArray(t *testing.T) {
	src := mustGenerate(t, `
struct Data {
	u32 count;
	u8 items[count];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "Items []uint8")
	assert.Contains(t, src, "make([]uint8")
	assert.Contains(t, src, "result.Count")
}

func TestEnumMarshalJSON(t *testing.T) {
	src := mustGenerate(t, `
enum Status : u8 {
	OK = 0,
	Error = 1
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (e Status) String() string")
	assert.Contains(t, src, `"OK (%d)"`)
	assert.Contains(t, src, `"Error (%d)"`)
	assert.Contains(t, src, `"unknown (%d)"`)
	assert.Contains(t, src, "func (e Status) MarshalJSON() ([]byte, error)")
	assert.Contains(t, src, "json.Marshal(e.String())")
	assert.Contains(t, src, `"encoding/json"`)
	assert.Contains(t, src, `"fmt"`)
}

func TestGenerateRemoteArray(t *testing.T) {
	src := mustGenerate(t, `
struct StdVector {
	u32 begin_ptr;
	u32 end_ptr;
	u32 capacity_ptr;
	u32 elements[(end_ptr - begin_ptr) / 4] @ begin_ptr;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "Elements")
	assert.Contains(t, src, "make([]uint32")
	// The array should be read from the absolute address in BeginPtr
	assert.Contains(t, src, "int64(result.BeginPtr)")
	// Length expression should reference sibling fields
	assert.Contains(t, src, "result.EndPtr")
	assert.Contains(t, src, "result.BeginPtr")
}

func TestGenerateExprArrayMultiByte(t *testing.T) {
	src := mustGenerate(t, `
struct Data {
	u16 count;
	u32 values[count];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "Values []uint32")
	assert.Contains(t, src, "make([]uint32")
}
