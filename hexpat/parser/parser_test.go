package parser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimpleStruct(t *testing.T) {
	input := `
struct Header {
    u32 magic;
    u16 version;
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	s, ok := f.Items[0].(StructDef)
	require.True(t, ok)
	assert.Equal(t, "Header", s.Name)
	assert.Len(t, s.Body, 2)
}

func TestParseEnum(t *testing.T) {
	input := `
enum Compression : u32 {
    BI_RGB,
    BI_RLE8,
    BI_RLE4,
    BI_BITFIELDS,
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	e, ok := f.Items[0].(EnumDef)
	require.True(t, ok)
	assert.Equal(t, "Compression", e.Name)
	assert.Len(t, e.Members, 4)
	assert.Equal(t, "BI_RGB", e.Members[0].Name)
}

func TestParseBitfield(t *testing.T) {
	input := `
bitfield Mode {
    x           : 3;
    w           : 3;
    r           : 3;
    sticky      : 1;
    sgid        : 1;
    suid        : 1;
    file_type   : 4;
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	bf, ok := f.Items[0].(BitfieldDef)
	require.True(t, ok)
	assert.Equal(t, "Mode", bf.Name)
	assert.Len(t, bf.Body, 7)
	assert.Equal(t, "x", bf.Body[0].Name)
}

func TestParseUsing(t *testing.T) {
	input := `using Offset = be u32;`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	u, ok := f.Items[0].(UsingDef)
	require.True(t, ok)
	assert.Equal(t, "Offset", u.Name)
	et, ok := u.Type.(EndianType)
	require.True(t, ok)
	assert.Equal(t, "be", et.Order)
}

func TestParsePragma(t *testing.T) {
	input := `
#pragma author WerWolv
#pragma description GNU Static library archive
#pragma endian little
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 3)

	p0, ok := f.Items[0].(Pragma)
	require.True(t, ok)
	assert.Equal(t, "author", p0.Key)
	assert.Equal(t, "WerWolv", p0.Value)
}

func TestParseImport(t *testing.T) {
	input := `
import std.string;
import std.mem;
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 2)

	imp, ok := f.Items[0].(Import)
	require.True(t, ok)
	assert.Equal(t, []string{"std", "string"}, imp.Path)
}

func TestParseVarDecl(t *testing.T) {
	input := `u32 value @ 0x100;`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	v, ok := f.Items[0].(VarDecl)
	require.True(t, ok)
	assert.Equal(t, "value", v.Name)
	assert.NotNil(t, v.Offset)
}

func TestParseArrayDecl(t *testing.T) {
	input := `char signature[8] @ 0x00;`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	v, ok := f.Items[0].(VarDecl)
	require.True(t, ok)
	assert.Equal(t, "signature", v.Name)
	assert.NotNil(t, v.Array)
}

func TestParseStructWithConditional(t *testing.T) {
	input := `
struct ARFile {
    char file_name[16];
    char modification_timestamp[12];
    u16 end_marker;

    if (end_marker == 0x0A60) {
        u8 data[10];
    }
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	s, ok := f.Items[0].(StructDef)
	require.True(t, ok)
	assert.Equal(t, "ARFile", s.Name)
	// Should have: file_name, modification_timestamp, end_marker, if-stmt
	assert.True(t, len(s.Body) >= 4, "expected at least 4 body items, got %d", len(s.Body))
}

func TestParseStructInheritance(t *testing.T) {
	input := `
struct Base {
    u32 size;
};

struct Child : Base {
    u32 extra;
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 2)

	child, ok := f.Items[1].(StructDef)
	require.True(t, ok)
	assert.Equal(t, "Child", child.Name)
	assert.Equal(t, "Base", child.Parent)
}

func TestParseFn(t *testing.T) {
	input := `
fn swap_32bit(u32 value) {
    return ((value >> 16) & 0xFFFF) | ((value & 0xFFFF) << 16);
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	fn, ok := f.Items[0].(FnDef)
	require.True(t, ok)
	assert.Equal(t, "swap_32bit", fn.Name)
	assert.Len(t, fn.Params, 1)
	assert.Equal(t, "value", fn.Params[0].Name)
}

func TestParseNamespace(t *testing.T) {
	input := `
namespace old_binary {
    using Time = u32;
    struct Header {
        u16 magic;
    };
}
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	ns, ok := f.Items[0].(NamespaceDef)
	require.True(t, ok)
	assert.Equal(t, "old_binary", ns.Name)
	assert.Len(t, ns.Items, 2)
}

func TestParseAttributes(t *testing.T) {
	input := `
struct Foo {
    u8 data[10] [[color("FF0000"), hidden]];
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	s, ok := f.Items[0].(StructDef)
	require.True(t, ok)
	require.Len(t, s.Body, 1)
	v, ok := s.Body[0].(VarDecl)
	require.True(t, ok)
	assert.Len(t, v.Attrs, 2)
	assert.Equal(t, "color", v.Attrs[0].Name)
	assert.Equal(t, "hidden", v.Attrs[1].Name)
}

func TestParseIfDef(t *testing.T) {
	input := `
#ifdef __IMHEX__
    import hex.core;
#endif
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	ifdef, ok := f.Items[0].(IfDef)
	require.True(t, ok)
	assert.Equal(t, "__IMHEX__", ifdef.Name)
	assert.False(t, ifdef.Negated)
	assert.Len(t, ifdef.Body, 1)
}

func TestParseMatch(t *testing.T) {
	input := `
struct Bitmap {
    u32 headerSize;
    match (headerSize) {
        (40):  u32 v1;
        (52):  u32 v2;
        (_):   u32 vDefault;
    }
};
`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	s, ok := f.Items[0].(StructDef)
	require.True(t, ok)
	// headerSize + match
	assert.True(t, len(s.Body) >= 2, "expected at least 2 body items, got %d", len(s.Body))
}

// --- Real .hexpat file tests ---

const patternsDir = "/home/vitaminmoo/repos/ImHex-Patterns/patterns"

func readHexpatFile(t *testing.T, name string) string {
	t.Helper()
	path := patternsDir + "/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("Skipping: %s not available: %v", path, err)
	}
	return string(data)
}

func TestParseARFile(t *testing.T) {
	input := readHexpatFile(t, "ar.hexpat")
	f, err := Parse(input)
	require.NoError(t, err)
	assert.NotEmpty(t, f.Items)

	// Should have pragmas, imports, struct ARFile, and variable declarations
	var hasStruct, hasPragma, hasImport bool
	for _, item := range f.Items {
		switch item.(type) {
		case StructDef:
			hasStruct = true
		case Pragma:
			hasPragma = true
		case Import:
			hasImport = true
		}
	}
	assert.True(t, hasPragma, "expected pragma")
	assert.True(t, hasImport, "expected import")
	assert.True(t, hasStruct, "expected struct")
}

func TestParseARCFile(t *testing.T) {
	input := readHexpatFile(t, "arc.hexpat")
	f, err := Parse(input)
	require.NoError(t, err)
	assert.NotEmpty(t, f.Items)

	var structCount int
	for _, item := range f.Items {
		if _, ok := item.(StructDef); ok {
			structCount++
		}
	}
	assert.Equal(t, 2, structCount, "expected Table and ARC structs")
}

func TestParseBMPFile(t *testing.T) {
	input := readHexpatFile(t, "bmp.hexpat")
	f, err := Parse(input)
	require.NoError(t, err)
	assert.NotEmpty(t, f.Items)

	var structCount, enumCount int
	for _, item := range f.Items {
		switch item.(type) {
		case StructDef:
			structCount++
		case EnumDef:
			enumCount++
		}
	}
	assert.True(t, structCount >= 7, "expected at least 7 structs, got %d", structCount)
	assert.Equal(t, 1, enumCount, "expected 1 enum")
}

func TestParseCPIOFile(t *testing.T) {
	input := readHexpatFile(t, "cpio.hexpat")
	f, err := Parse(input)
	require.NoError(t, err)
	assert.NotEmpty(t, f.Items)

	// Should have namespace with structs, bitfield, using, fn
	var hasNamespace bool
	for _, item := range f.Items {
		if _, ok := item.(NamespaceDef); ok {
			hasNamespace = true
		}
	}
	assert.True(t, hasNamespace, "expected namespace")
}

func TestParseBulkFiles(t *testing.T) {
	entries, err := os.ReadDir(patternsDir)
	if err != nil {
		t.Skip("ImHex patterns not available")
	}

	var passed, failed int
	var failures []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < 8 || name[len(name)-7:] != ".hexpat" {
			continue
		}
		data, readErr := os.ReadFile(patternsDir + "/" + name)
		if readErr != nil {
			continue
		}
		_, err := Parse(string(data))
		if err != nil {
			failed++
			failures = append(failures, name+": "+err.Error())
		} else {
			passed++
		}
	}
	t.Logf("Parsed %d/%d files successfully", passed, passed+failed)
	for _, f := range failures {
		t.Logf("  FAIL: %s", f)
	}
	// We want at least 80% success rate
	total := passed + failed
	if total > 0 {
		rate := float64(passed) / float64(total) * 100
		t.Logf("Success rate: %.1f%%", rate)
	}
}
