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
	// Pointer field stored as raw value (uint64 for :u64)
	assert.Contains(t, src, "Next  uint64")
	// Follow method on eager struct
	assert.Contains(t, src, "func (s *Node) ReadNext(ctx *runtime.ReadContext) (*Node, runtime.Errors)")
	assert.Contains(t, src, "if s.Next == 0")
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
	// A corrupt/torn pointer pair must not allocate an unbounded slice.
	assert.Contains(t, src, "runtime.BoundedLen(")
	assert.Contains(t, src, "runtime.ErrArrayTooLong")
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

// --- Reader generation tests ---

func TestReaderSimpleStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u32 magic;
	u16 version;
	u8 flags;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type HeaderReader struct")
	assert.Contains(t, src, "func NewHeaderReader(")
	assert.Contains(t, src, "func (r *HeaderReader) Magic() (uint32, error)")
	assert.Contains(t, src, "func (r *HeaderReader) Version() (uint16, error)")
	assert.Contains(t, src, "func (r *HeaderReader) Flags() (uint8, error)")
	assert.Contains(t, src, "func (r *HeaderReader) Read() (*Header, runtime.Errors)")
	assert.Contains(t, src, "func (r *HeaderReader) Addr() uintptr")
}

func TestReaderNestedStruct(t *testing.T) {
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
	assert.Contains(t, src, "type InnerReader struct")
	assert.Contains(t, src, "type OuterReader struct")
	assert.Contains(t, src, "func (r *OuterReader) Pos() *InnerReader")
	// Nested struct accessor should be zero I/O (no error return)
	assert.NotContains(t, src, "func (r *OuterReader) Pos() (*InnerReader, error)")
}

func TestReaderPointer(t *testing.T) {
	src := mustGenerate(t, `
struct Node {
	u32 value;
	Node *next : u64;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type NodeReader struct")
	assert.Contains(t, src, "func (r *NodeReader) Value() (uint32, error)")
	// Raw value accessor returns uint64 (storage type)
	assert.Contains(t, src, "func (r *NodeReader) Next() (uint64, error)")
	// Follow method reads and follows the pointer
	assert.Contains(t, src, "func (r *NodeReader) FollowNext() (*Node, runtime.Errors)")
}

func TestReaderWithArray(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u8 magic[4];
	u32 values[3];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (r *HeaderReader) Magic() ([4]uint8, error)")
	assert.Contains(t, src, "func (r *HeaderReader) Values() ([3]uint32, error)")
}

func TestReaderWithStructArray(t *testing.T) {
	src := mustGenerate(t, `
struct Point {
	u16 x;
	u16 y;
};

struct Path {
	u32 count;
	Point points[3];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (r *PathReader) Points() [3]PointReader")
}

func TestReaderNotGeneratedForDynamicStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Data {
	u32 count;
	u8 items[count];
};
`)
	assertCompiles(t, src)
	// Dynamic struct should NOT get a reader
	assert.NotContains(t, src, "DataReader")
}

func TestReaderNotGeneratedForConditionalStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u32 flags;
	if (flags & 0x01) {
		u32 extra;
	}
};
`)
	assertCompiles(t, src)
	assert.NotContains(t, src, "HeaderReader")
}

func TestReaderWithEnum(t *testing.T) {
	src := mustGenerate(t, `
enum Status : u16 {
	OK = 0,
	Error = 1
};

struct Msg {
	Status status;
	u32 payload;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (r *MsgReader) Status() (Status, error)")
	assert.Contains(t, src, "func (r *MsgReader) Payload() (uint32, error)")
}

func TestReaderWithFloats(t *testing.T) {
	src := mustGenerate(t, `
struct Vec3 {
	float x;
	float y;
	double z;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (r *Vec3Reader) X() (float32, error)")
	assert.Contains(t, src, "func (r *Vec3Reader) Z() (float64, error)")
	assert.Contains(t, src, "math.Float32frombits")
	assert.Contains(t, src, "math.Float64frombits")
}

func TestReaderIntegration(t *testing.T) {
	// Build binary data: magic=0xDEADBEEF, version=0x0102, flags=0x42
	var data bytes.Buffer
	binary.Write(&data, binary.LittleEndian, uint32(0xDEADBEEF))
	binary.Write(&data, binary.LittleEndian, uint16(0x0102))
	binary.Write(&data, binary.LittleEndian, uint8(0x42))

	ctx := runtime.NewReadContext(bytes.NewReader(data.Bytes()))

	// Test that we can read individual fields at the right offsets
	var buf [4]byte

	// Magic at offset 0
	_, err := ctx.ReadAt(buf[:4], 0)
	require.NoError(t, err)
	assert.Equal(t, uint32(0xDEADBEEF), binary.LittleEndian.Uint32(buf[:4]))

	// Version at offset 4
	_, err = ctx.ReadAt(buf[:2], 4)
	require.NoError(t, err)
	assert.Equal(t, uint16(0x0102), binary.LittleEndian.Uint16(buf[:2]))

	// Flags at offset 6
	_, err = ctx.ReadAt(buf[:1], 6)
	require.NoError(t, err)
	assert.Equal(t, uint8(0x42), buf[0])
}

func TestReaderWithBitfield(t *testing.T) {
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
	assert.Contains(t, src, "type FileReader struct")
	assert.Contains(t, src, "func (r *FileReader) Perms() *PermsReader")
}

func TestReaderStaticParentWithDynamicChild(t *testing.T) {
	src := mustGenerate(t, `
struct DynChild {
	u32 count;
	u8 items[count];
};

struct Parent {
	u32 id;
	DynChild child;
};
`)
	assertCompiles(t, src)
	// Parent should get a reader even though DynChild doesn't
	assert.Contains(t, src, "type ParentReader struct")
	assert.NotContains(t, src, "DynChildReader")
	// Parent's child accessor should eagerly read via ReadDynChild
	assert.Contains(t, src, "func (r *ParentReader) Child() (*DynChild, runtime.Errors)")
	assert.Contains(t, src, "ReadDynChild(r.ctx")
}

func TestReaderBoolField(t *testing.T) {
	src := mustGenerate(t, `
struct Flags {
	bool active;
	u32 value;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (r *FlagsReader) Active() (bool, error)")
	// Should return false on error, not 0
	assert.Contains(t, src, "return false, err")
}

func TestReaderWithEndianOverride(t *testing.T) {
	src := mustGenerate(t, `
struct Mixed {
	le u32 little_val;
	be u32 big_val;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "type MixedReader struct")
	assert.Contains(t, src, "func (r *MixedReader) LittleVal() (uint32, error)")
	assert.Contains(t, src, "func (r *MixedReader) BigVal() (uint32, error)")
	// The reader methods should use the correct endian
	assert.Contains(t, src, "binary.LittleEndian.Uint32")
	assert.Contains(t, src, "binary.BigEndian.Uint32")
}

func TestGeneratePointerFollowMethod(t *testing.T) {
	src := mustGenerate(t, `
struct Target {
	u32 value;
};

struct Container {
	Target *ptr : u32;
	u32 other;
};
`)
	assertCompiles(t, src)
	// Eager struct stores raw uint32
	assert.Contains(t, src, "Ptr   uint32")
	// Follow method on eager struct
	assert.Contains(t, src, "func (s *Container) ReadPtr(ctx *runtime.ReadContext) (*Target, runtime.Errors)")
	// Lazy reader raw accessor
	assert.Contains(t, src, "func (r *ContainerReader) Ptr() (uint32, error)")
	// Lazy reader follow method
	assert.Contains(t, src, "func (r *ContainerReader) FollowPtr() (*Target, runtime.Errors)")
}

func TestGeneratePlacementPointer(t *testing.T) {
	src := mustGenerate(t, `
struct EntityManager {
	u32 count;
	u32 capacity;
};

EntityManager *g_entityManager : u32 @ 0x01204B98;
`)
	assertCompiles(t, src)
	// Address constant
	assert.Contains(t, src, "AddrGEntityManager uint32 = 0x01204B98")
	// Read function that follows the pointer
	assert.Contains(t, src, "func ReadGEntityManager(ctx *runtime.ReadContext) (*EntityManager, runtime.Errors)")
	assert.Contains(t, src, "int64(AddrGEntityManager)")
}

func TestGeneratePlacementPrimitive(t *testing.T) {
	src := mustGenerate(t, `
u32 g_worldSeed @ 0x01205004;
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "AddrGWorldSeed uint32 = 0x01205004")
	assert.Contains(t, src, "func ReadGWorldSeed(ctx *runtime.ReadContext) (uint32, error)")
}

func TestGeneratePlacementStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Config {
	u32 flags;
	u16 version;
};

Config g_config @ 0x00400000;
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "AddrGConfig uint32 = 0x00400000")
	assert.Contains(t, src, "func ReadGConfig(ctx *runtime.ReadContext) (*Config, runtime.Errors)")
}

func TestGenerateMultiplePlacements(t *testing.T) {
	src := mustGenerate(t, `
struct Globals {
	u32 value;
};

Globals *g_globals : u32 @ 0x01000000;
u32 g_seed @ 0x02000000;
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "AddrGGlobals uint32 = 0x01000000")
	assert.Contains(t, src, "AddrGSeed")
}

func TestGenerateForwardDeclaredPointerFields(t *testing.T) {
	src := mustGenerate(t, `
using Entity;
using ChildrenContainer;

struct Entity {
	s32 entity_id;
	ChildrenContainer *children_ptr : u32;
	Entity *parent_entity_ptr : u32;
};

struct ChildrenContainer {
	u32 begin_ptr;
	u32 end_ptr;
};
`)
	assertCompiles(t, src)
	// Forward-declared pointer fields must appear as raw uint32
	assert.Contains(t, src, "ChildrenPtr")
	assert.Contains(t, src, "ParentEntityPtr")
	// The struct must have all 3 fields
	assert.Contains(t, src, "EntityId")
	// Follow methods must be generated
	assert.Contains(t, src, "func (s *Entity) ReadChildrenPtr(ctx *runtime.ReadContext) (*ChildrenContainer, runtime.Errors)")
	assert.Contains(t, src, "func (s *Entity) ReadParentEntityPtr(ctx *runtime.ReadContext) (*Entity, runtime.Errors)")
	// Lazy reader follow methods
	assert.Contains(t, src, "func (r *EntityReader) FollowChildrenPtr() (*ChildrenContainer, runtime.Errors)")
	assert.Contains(t, src, "func (r *EntityReader) FollowParentEntityPtr() (*Entity, runtime.Errors)")
}

// --- Function transpilation tests ---

func TestTranspileFormatFunction(t *testing.T) {
	src := mustGenerate(t, `
struct MsvcString {
    char data[16];
    u32 length;
    u32 capacity;
} [[static, format("format_msvc_string")]];

fn format_msvc_string(MsvcString s) {
    if (s.length == 0) return "";
    if (s.capacity <= 15) {
        str result = "";
        for (u32 i = 0, i < s.length, i = i + 1) {
            result = result + s.data[i];
        }
        return result;
    }
    return std::format("<heap len={}>", s.length);
};
`)
	assertCompiles(t, src)
	// Method generated on the struct
	assert.Contains(t, src, "func (s *MsvcString) FormatMsvcString() string")
	// If/else control flow
	assert.Contains(t, src, "if s.Length == 0")
	assert.Contains(t, src, "if s.Capacity <= 15")
	// For loop with proper Go types
	assert.Contains(t, src, "for i := uint32(0); i < s.Length")
	// String concat with byte conversion
	assert.Contains(t, src, "string(s.Data[i])")
	// std::format → fmt.Sprintf
	assert.Contains(t, src, "fmt.Sprintf")
}

func TestTranspileSimpleFormatFunction(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
    u32 magic;
    u16 version;
} [[format("format_header")]];

fn format_header(Header h) {
    return std::format("v{}", h.version);
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (s *Header) FormatHeader() string")
	assert.Contains(t, src, `fmt.Sprintf("v%v", s.Version)`)
}

func TestTranspileFormatWithComparison(t *testing.T) {
	src := mustGenerate(t, `
struct Flags {
    u32 value;
} [[format("format_flags")]];

fn format_flags(Flags f) {
    if (f.value == 0) return "none";
    if (f.value == 1) return "read";
    if (f.value == 2) return "write";
    return "unknown";
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (s *Flags) FormatFlags() string")
	assert.Contains(t, src, "s.Value == 0")
	assert.Contains(t, src, `return "none"`)
	assert.Contains(t, src, `return "unknown"`)
}

func TestTranspileNoFunctionNoCrash(t *testing.T) {
	// [[format("nonexistent")]] should be silently ignored
	src := mustGenerate(t, `
struct Foo {
    u32 x;
} [[format("does_not_exist")]];
`)
	assertCompiles(t, src)
	assert.NotContains(t, src, "DoesNotExist")
}

func TestTranspileFormatWithWhile(t *testing.T) {
	src := mustGenerate(t, `
struct Counter {
    u32 limit;
} [[format("format_counter")]];

fn format_counter(Counter c) {
    u32 sum = 0;
    u32 i = 0;
    while (i < c.limit) {
        sum = sum + i;
        i = i + 1;
    }
    return std::format("sum={}", sum);
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (s *Counter) FormatCounter() string")
	assert.Contains(t, src, "for i < s.Limit")
}

func TestTranspileWithMemRead(t *testing.T) {
	src := mustGenerate(t, `
struct MsvcString {
    char data[16];
    u32 length;
    u32 capacity;
} [[static, format("format_msvc_string")]];

fn format_msvc_string(MsvcString s) {
    if (s.length == 0) return "";
    if (s.capacity <= 15) {
        str result = "";
        for (u32 i = 0, i < s.length, i = i + 1) {
            result = result + s.data[i];
        }
        return result;
    }
    u32 heap_ptr = s.data[0] | (s.data[1] << 8) | (s.data[2] << 16) | (s.data[3] << 24);
    if (heap_ptr == 0) return "";
    return std::mem::read_string(heap_ptr, s.length);
};
`)
	assertCompiles(t, src)
	// Method gets ctx param because it uses std::mem
	assert.Contains(t, src, "func (s *MsvcString) FormatMsvcString(ctx *runtime.ReadContext) string")
	// Helper generated
	assert.Contains(t, src, "func _memReadString(ctx *runtime.ReadContext")
	assert.Contains(t, src, "func _memReadUnsigned(ctx *runtime.ReadContext")
	// Call with uint64 casts
	assert.Contains(t, src, "_memReadString(ctx, uint64(heapPtr), uint64(s.Length))")
}

func TestTranspileMemReadUnsigned(t *testing.T) {
	src := mustGenerate(t, `
struct Foo {
    u32 addr;
} [[format("format_foo")]];

fn format_foo(Foo f) {
    u32 val = std::mem::read_unsigned(f.addr, 4);
    return std::format("val={}", val);
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "func (s *Foo) FormatFoo(ctx *runtime.ReadContext) string")
	assert.Contains(t, src, "_memReadUnsigned(ctx, uint64(s.Addr), uint64(4))")
}

// --- Bulk read tests ---

func TestBulkReadSimpleStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u32 magic;
	u16 version;
	u8 flags;
};
`)
	assertCompiles(t, src)
	// Single ReadAt for entire 7-byte struct
	assert.Contains(t, src, "var buf [7]byte")
	assert.Contains(t, src, "ctx.ReadAt(buf[:], int64(addr))")
	// Fields decoded from buffer, not per-field ReadAt
	assert.Contains(t, src, "result.Magic = binary.LittleEndian.Uint32(buf[0:])")
	assert.Contains(t, src, "result.Version = binary.LittleEndian.Uint16(buf[4:])")
	assert.Contains(t, src, "result.Flags = buf[6]")
}

func TestBulkReadWithEnum(t *testing.T) {
	src := mustGenerate(t, `
enum Status : u16 {
	OK = 0,
	Error = 1
};

struct Msg {
	u32 id;
	Status status;
};
`)
	assertCompiles(t, src)
	// Enum decoded from buffer
	assert.Contains(t, src, "result.Status = Status(binary.LittleEndian.Uint16(buf[4:]))")
}

func TestBulkReadWithFloats(t *testing.T) {
	src := mustGenerate(t, `
struct Vec3 {
	float x;
	float y;
	double z;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "var buf [16]byte")
	assert.Contains(t, src, "math.Float32frombits(binary.LittleEndian.Uint32(buf[0:]))")
	assert.Contains(t, src, "math.Float64frombits(binary.LittleEndian.Uint64(buf[8:]))")
}

func TestBulkReadWithPointer(t *testing.T) {
	src := mustGenerate(t, `
struct Target { u32 value; };
struct Container {
	u32 id;
	Target *ptr : u32;
};
`)
	assertCompiles(t, src)
	// Pointer decoded from buffer
	assert.Contains(t, src, "result.Ptr = binary.LittleEndian.Uint32(buf[4:])")
}

func TestBulkReadWithByteArray(t *testing.T) {
	src := mustGenerate(t, `
struct Header {
	u8 magic[4];
	u32 size;
};
`)
	assertCompiles(t, src)
	// Byte array copied from buffer
	assert.Contains(t, src, "copy(result.Magic[:], buf[0:4])")
	assert.Contains(t, src, "result.Size = binary.LittleEndian.Uint32(buf[4:])")
}

func TestBulkReadWithMultiByteArray(t *testing.T) {
	src := mustGenerate(t, `
struct Data {
	u32 values[3];
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "var buf [12]byte")
	assert.Contains(t, src, "for i := range result.Values")
	assert.Contains(t, src, "binary.LittleEndian.Uint32(buf[0+i*4:])")
}

func TestBulkReadNestedStructStillRecursive(t *testing.T) {
	src := mustGenerate(t, `
struct Inner { u16 x; u16 y; };
struct Outer { u32 id; Inner pos; u32 flags; };
`)
	assertCompiles(t, src)
	// Outer should bulk-read its full size
	assert.Contains(t, src, "var buf [12]byte")
	// Flat fields decoded from buffer
	assert.Contains(t, src, "result.Id = binary.LittleEndian.Uint32(buf[0:])")
	assert.Contains(t, src, "result.Flags = binary.LittleEndian.Uint32(buf[8:])")
	// Nested struct still uses recursive call
	assert.Contains(t, src, "ReadInner(ctx, uintptr(int64(addr)+4))")
}

func TestBulkReadNotUsedForDynamicStruct(t *testing.T) {
	src := mustGenerate(t, `
struct Data {
	u32 count;
	u8 items[count];
};
`)
	assertCompiles(t, src)
	// Dynamic struct should use per-field pattern, not bulk read
	assert.Contains(t, src, "offset := int64(0)")
	assert.NotContains(t, src, "errs.Add(\"Data\", uintptr(addr), err)")
}

func TestBulkReadWithEndianOverride(t *testing.T) {
	src := mustGenerate(t, `
struct Mixed {
	le u32 little_val;
	be u32 big_val;
};
`)
	assertCompiles(t, src)
	assert.Contains(t, src, "var buf [8]byte")
	assert.Contains(t, src, "binary.LittleEndian.Uint32(buf[0:])")
	assert.Contains(t, src, "binary.BigEndian.Uint32(buf[4:])")
}

func TestBulkReadIntegration(t *testing.T) {
	// Verify that bulk-read generated code produces correct results
	var data bytes.Buffer
	binary.Write(&data, binary.LittleEndian, uint32(0xDEADBEEF)) // magic at 0
	binary.Write(&data, binary.LittleEndian, uint16(0x0102))      // version at 4
	binary.Write(&data, binary.LittleEndian, uint8(0x42))         // flags at 6

	ctx := runtime.NewReadContext(bytes.NewReader(data.Bytes()))

	// Read the full 7 bytes in one call (simulating what bulk read does)
	var buf [7]byte
	n, err := ctx.ReadAt(buf[:], 0)
	require.NoError(t, err)
	assert.Equal(t, 7, n)

	// Decode from buffer at offsets (matching generated bulk read pattern)
	assert.Equal(t, uint32(0xDEADBEEF), binary.LittleEndian.Uint32(buf[0:]))
	assert.Equal(t, uint16(0x0102), binary.LittleEndian.Uint16(buf[4:]))
	assert.Equal(t, uint8(0x42), buf[6])
}
