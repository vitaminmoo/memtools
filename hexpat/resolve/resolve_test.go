package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/hexpat/parser"
)

func mustParse(t *testing.T, src string) *parser.File {
	t.Helper()
	file, err := parser.Parse(src)
	require.NoError(t, err, "parse failed")
	return file
}

func TestResolveSimpleStruct(t *testing.T) {
	file := mustParse(t, `
struct Header {
	u32 magic;
	u16 version;
	u8 flags;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	require.Len(t, pkg.Structs, 1)
	st := pkg.Structs[0]
	assert.Equal(t, "Header", st.Name)
	fields := st.Fields()
	require.Len(t, fields, 3)

	assert.Equal(t, "Magic", fields[0].Name)
	assert.Equal(t, "uint32", fields[0].Type.GoType)
	assert.Equal(t, 0, fields[0].Offset)

	assert.Equal(t, "Version", fields[1].Name)
	assert.Equal(t, "uint16", fields[1].Type.GoType)
	assert.Equal(t, 4, fields[1].Offset)

	assert.Equal(t, "Flags", fields[2].Name)
	assert.Equal(t, "uint8", fields[2].Type.GoType)
	assert.Equal(t, 6, fields[2].Offset)

	assert.Equal(t, 7, st.Size)
}

func TestResolveEnum(t *testing.T) {
	file := mustParse(t, `
enum Compression : u32 {
	None = 0,
	RLE8 = 1,
	RLE4 = 2
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Enums, 1)

	et := pkg.Enums[0]
	assert.Equal(t, "Compression", et.Name)
	assert.Equal(t, "uint32", et.UnderlyingType.GoType)
	require.Len(t, et.Members, 3)
	assert.Equal(t, "None", et.Members[0].Name)
	assert.Equal(t, int64(0), et.Members[0].Value)
	assert.Equal(t, "RLE8", et.Members[1].Name)
	assert.Equal(t, int64(1), et.Members[1].Value)
	assert.Equal(t, "RLE4", et.Members[2].Name)
	assert.Equal(t, int64(2), et.Members[2].Value)
}

func TestResolveEnumAutoIncrement(t *testing.T) {
	file := mustParse(t, `
enum Status : u8 {
	OK,
	Warning,
	Error
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Enums, 1)

	et := pkg.Enums[0]
	assert.Equal(t, int64(0), et.Members[0].Value)
	assert.Equal(t, int64(1), et.Members[1].Value)
	assert.Equal(t, int64(2), et.Members[2].Value)
}

func TestResolveEnumInStruct(t *testing.T) {
	file := mustParse(t, `
enum Compression : u32 {
	None = 0,
	RLE8 = 1
};

struct Header {
	u32 magic;
	Compression compression;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, KindEnum, fields[1].Type.Kind)
	assert.Equal(t, "Compression", fields[1].Type.GoType)
	assert.Equal(t, 4, fields[1].Offset)
	assert.Equal(t, 8, st.Size)
}

func TestResolvePragmaEndian(t *testing.T) {
	file := mustParse(t, `
#pragma endian big
struct Foo {
	u32 x;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	assert.Equal(t, BigEndian, pkg.Endian)
}

func TestResolveAtOffset(t *testing.T) {
	file := mustParse(t, `
struct Sparse {
	u32 magic @ 0x00;
	u16 version @ 0x10;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	assert.Equal(t, 0, fields[0].Offset)
	assert.Equal(t, 16, fields[1].Offset)
	assert.Equal(t, 18, st.Size)
}

func TestResolveArray(t *testing.T) {
	file := mustParse(t, `
struct Header {
	u8 magic[4];
	u32 values[3];
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)

	assert.Equal(t, KindArray, fields[0].Type.Kind)
	assert.Equal(t, "[4]uint8", fields[0].Type.GoType)
	assert.Equal(t, 4, fields[0].Type.Size)

	assert.Equal(t, KindArray, fields[1].Type.Kind)
	assert.Equal(t, "[3]uint32", fields[1].Type.GoType)
	assert.Equal(t, 12, fields[1].Type.Size)
	assert.Equal(t, 4, fields[1].Offset)
}

func TestResolvePadding(t *testing.T) {
	file := mustParse(t, `
struct Padded {
	u32 magic;
	padding[4];
	u32 data;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, 0, fields[0].Offset)
	assert.Equal(t, 8, fields[1].Offset)
	assert.Equal(t, 12, st.Size)
}

func TestResolveNestedStruct(t *testing.T) {
	file := mustParse(t, `
struct Inner {
	u16 x;
	u16 y;
};

struct Outer {
	u32 id;
	Inner pos;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 2)

	// Topo sort: Inner before Outer
	assert.Equal(t, "Inner", pkg.Structs[0].Name)
	assert.Equal(t, "Outer", pkg.Structs[1].Name)

	outer := pkg.Structs[1]
	fields := outer.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, KindStruct, fields[1].Type.Kind)
	assert.Equal(t, "Inner", fields[1].Type.GoType)
	assert.Equal(t, 4, fields[1].Offset)
	assert.Equal(t, 8, outer.Size)
}

func TestResolvePointer(t *testing.T) {
	file := mustParse(t, `
struct Node {
	u32 value;
	Node *next : u64;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, KindPointer, fields[1].Type.Kind)
	assert.Equal(t, 8, fields[1].Type.Size)
	assert.Equal(t, 12, st.Size)
}

func TestResolveUsing(t *testing.T) {
	file := mustParse(t, `
using Offset = u32;

struct Header {
	Offset start;
	Offset end;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, "uint32", fields[0].Type.GoType)
	assert.Equal(t, "uint32", fields[1].Type.GoType)
}

func TestResolveToPascalCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello_world", "HelloWorld"},
		{"my_struct", "MyStruct"},
		{"Already", "Already"},
		{"u32", "U32"},
		{"", ""},
		{"a_b_c", "ABC"},
		{"type", "Type_"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, toPascalCase(tt.input), "toPascalCase(%q)", tt.input)
	}
}

func TestResolveInheritance(t *testing.T) {
	file := mustParse(t, `
struct Base {
	u32 id;
	u16 flags;
};

struct Derived : Base {
	u32 data;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	var derived *StructType
	for _, st := range pkg.Structs {
		if st.Name == "Derived" {
			derived = st
		}
	}
	require.NotNil(t, derived)
	fields := derived.Fields()
	require.Len(t, fields, 3)
	assert.Equal(t, "Id", fields[0].Name)
	assert.Equal(t, "Flags", fields[1].Name)
	assert.Equal(t, "Data", fields[2].Name)
	assert.Equal(t, 6, fields[2].Offset)
}

func TestResolveSnakeCaseFields(t *testing.T) {
	file := mustParse(t, `
struct my_struct {
	u32 field_one;
	u16 field_two;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	st := pkg.Structs[0]
	fields := st.Fields()
	assert.Equal(t, "MyStruct", st.Name)
	assert.Equal(t, "FieldOne", fields[0].Name)
	assert.Equal(t, "FieldTwo", fields[1].Name)
}

func TestResolveEndianOverride(t *testing.T) {
	file := mustParse(t, `
struct Mixed {
	le u32 little_val;
	be u32 big_val;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, LittleEndian, fields[0].Type.Endian)
	assert.Equal(t, BigEndian, fields[1].Type.Endian)
}

func TestResolveFloats(t *testing.T) {
	file := mustParse(t, `
struct Floats {
	float x;
	double y;
	f32 z;
	f64 w;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 4)
	assert.Equal(t, "float32", fields[0].Type.GoType)
	assert.Equal(t, 4, fields[0].Type.Size)
	assert.Equal(t, "float64", fields[1].Type.GoType)
	assert.Equal(t, 8, fields[1].Type.Size)
	assert.Equal(t, "float32", fields[2].Type.GoType)
	assert.Equal(t, "float64", fields[3].Type.GoType)
}

// --- New tests for Phase 2 features ---

func TestResolveUnion(t *testing.T) {
	file := mustParse(t, `
union Value {
	u32 as_int;
	float as_float;
	u8 as_bytes[4];
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	assert.True(t, st.IsUnion)
	fields := st.Fields()
	require.Len(t, fields, 3)

	// All fields at offset 0
	assert.Equal(t, 0, fields[0].Offset)
	assert.Equal(t, 0, fields[1].Offset)
	assert.Equal(t, 0, fields[2].Offset)

	// Size = max field size = 4
	assert.Equal(t, 4, st.Size)
}

func TestResolveBitfield(t *testing.T) {
	file := mustParse(t, `
bitfield Flags {
	a : 1;
	b : 3;
	padding : 4;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Bitfields, 1)

	bt := pkg.Bitfields[0]
	assert.Equal(t, "Flags", bt.Name)
	assert.Equal(t, 8, bt.TotalBits)
	assert.Equal(t, "uint8", bt.Underlying.GoType)
	require.Len(t, bt.Fields, 2) // padding is skipped

	assert.Equal(t, "A", bt.Fields[0].Name)
	assert.Equal(t, 1, bt.Fields[0].Bits)
	assert.Equal(t, 0, bt.Fields[0].BitOffset)
	assert.Equal(t, "bool", bt.Fields[0].GoType)

	assert.Equal(t, "B", bt.Fields[1].Name)
	assert.Equal(t, 3, bt.Fields[1].Bits)
	assert.Equal(t, 1, bt.Fields[1].BitOffset)
	assert.Equal(t, "uint8", bt.Fields[1].GoType)
}

func TestResolveBitfieldInStruct(t *testing.T) {
	file := mustParse(t, `
bitfield Flags {
	readable : 1;
	writable : 1;
	padding : 6;
};

struct Entry {
	u32 id;
	Flags flags;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)
	assert.Equal(t, KindBitfield, fields[1].Type.Kind)
	assert.Equal(t, "Flags", fields[1].Type.GoType)
	assert.Equal(t, 1, fields[1].Type.Size) // underlying u8
	assert.Equal(t, 5, st.Size)             // 4 + 1
}

func TestResolveConditional(t *testing.T) {
	file := mustParse(t, `
struct Header {
	u32 flags;
	if (flags & 0x01) {
		u32 extra;
	}
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	assert.True(t, st.HasConditionals())
	assert.Equal(t, -1, st.Size) // dynamic

	fields := st.Fields()
	require.Len(t, fields, 2) // flags + extra
	assert.Equal(t, "Flags", fields[0].Name)
	assert.Equal(t, "Extra", fields[1].Name)

	// Check the conditional member
	require.Len(t, st.Members, 2) // FieldMember + ConditionalMember
	cm, ok := st.Members[1].(*ConditionalMember)
	require.True(t, ok)
	require.Len(t, cm.Branches, 1)
	assert.Contains(t, cm.Branches[0].Cond, "result.Flags")
	assert.Contains(t, cm.Branches[0].Cond, "0x01")
}

func TestResolveConditionalElse(t *testing.T) {
	file := mustParse(t, `
struct Header {
	u8 type;
	if (type == 1) {
		u32 value_a;
	} else {
		u16 value_b;
	}
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 3) // type + value_a + value_b

	cm, ok := st.Members[1].(*ConditionalMember)
	require.True(t, ok)
	require.Len(t, cm.Branches, 2)
	assert.NotEmpty(t, cm.Branches[0].Cond) // if
	assert.Empty(t, cm.Branches[1].Cond)    // else
}

func TestResolveExprArray(t *testing.T) {
	file := mustParse(t, `
struct Data {
	u32 count;
	u8 items[count];
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 2)

	arr := fields[1].Type
	assert.Equal(t, KindArray, arr.Kind)
	assert.Equal(t, -1, arr.Array.Length)
	assert.Equal(t, "result.Count", arr.Array.LengthExpr)
	assert.Equal(t, "[]uint8", arr.GoType)
	assert.Equal(t, -1, st.Size) // dynamic
}

func TestResolveUnresolvableTypePadding(t *testing.T) {
	file := mustParse(t, `
struct ELFHeader {
	type::Magic<"\x7fELF"> magic;
	u8 ei_class;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 1) // only ei_class (magic is unresolvable)
	assert.Equal(t, "EiClass", fields[0].Name)
	assert.Equal(t, 4, fields[0].Offset) // offset 4, not 0
	assert.Equal(t, 5, st.Size)          // 4 (padding) + 1 (u8)
}

func TestExprToGo(t *testing.T) {
	fieldMap := map[string]string{
		"flags": "Flags",
		"count": "Count",
	}

	tests := []struct {
		name string
		expr parser.Expr
		want string
	}{
		{"number", parser.NumberLit{Value: 42}, "42"},
		{"hex number", parser.NumberLit{Value: 255, Raw: "0xFF"}, "0xFF"},
		{"bool true", parser.BoolLit{Value: true}, "true"},
		{"bool false", parser.BoolLit{Value: false}, "false"},
		{"field ref", parser.Ident{Name: "flags"}, "result.Flags"},
		{"unknown ident", parser.Ident{Name: "MAGIC"}, "MAGIC"},
		{"binary and", parser.BinaryExpr{
			Op:    "&",
			Left:  parser.Ident{Name: "flags"},
			Right: parser.NumberLit{Value: 1},
		}, "(result.Flags & 1)"},
		{"unary not", parser.UnaryExpr{
			Op:      "!",
			Operand: parser.Ident{Name: "flags"},
			Prefix:  true,
		}, "(!result.Flags)"},
		{"bitwise not", parser.UnaryExpr{
			Op:      "~",
			Operand: parser.Ident{Name: "flags"},
			Prefix:  true,
		}, "(^result.Flags)"},
		{"member access", parser.MemberAccess{
			Object: parser.Ident{Name: "flags"},
			Member: "field_name",
		}, "result.Flags.FieldName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exprToGo(tt.expr, fieldMap)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveDynamicAtOffset(t *testing.T) {
	file := mustParse(t, `
struct StdVector {
	u32 begin_ptr;
	u32 end_ptr;
	u32 capacity_ptr;
	u32 elements[(end_ptr - begin_ptr) / 4] @ begin_ptr;
};
`)
	pkg, err := Resolve(file)
	require.NoError(t, err)
	require.Len(t, pkg.Structs, 1)

	st := pkg.Structs[0]
	fields := st.Fields()
	require.Len(t, fields, 4)

	// Inline fields have normal offsets
	assert.Equal(t, 0, fields[0].Offset)
	assert.Equal(t, 4, fields[1].Offset)
	assert.Equal(t, 8, fields[2].Offset)

	// Remote field has OffsetExpr and Offset -1
	assert.Equal(t, -1, fields[3].Offset)
	assert.Equal(t, "result.BeginPtr", fields[3].OffsetExpr)
	assert.Equal(t, KindArray, fields[3].Type.Kind)
	assert.Equal(t, -1, fields[3].Type.Array.Length)
	assert.Equal(t, "((result.EndPtr - result.BeginPtr) / 4)", fields[3].Type.Array.LengthExpr)

	// Struct is dynamic due to remote field
	assert.True(t, st.HasDynamicFields())
	assert.Equal(t, -1, st.Size)
}
