package hexpat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Category A: Array initializers with braces { }
// Affects: q3demo.hexpat, kindle_update.hexpat
// =============================================================================

func TestParseArrayInitializerBraces(t *testing.T) {
	input := `u16 table[4] = {1, 2, 3, 4};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	v, ok := f.Items[0].(VarDecl)
	require.True(t, ok)
	assert.Equal(t, "table", v.Name)
	assert.NotNil(t, v.Array)
	assert.NotNil(t, v.Init)
}

func TestParseConstArrayInitializer(t *testing.T) {
	// kindle_update.hexpat: const u8 gtop[256] = { 0xa7, 0xb7, ... };
	input := `const u8 gtop[3] = { 0xa7, 0xb7, 0x87 };`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseLargeArrayInitializer(t *testing.T) {
	// q3demo.hexpat has a massive inline array: u16 huffdecode[2048]={...};
	input := `u16 huffdecode[8]={2512,2182,512,2763,1859,2808,512,2360};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)
}

// =============================================================================
// Category B: Default parameter values in functions
// Affects: dos.hexpat, parquet.hexpat
// =============================================================================

func TestParseFnDefaultParam(t *testing.T) {
	// dos.hexpat: fn formatNumber(u32 num, str msg="")
	input := `fn formatNumber(u32 num, str msg = "") {
    return num;
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	fn, ok := f.Items[0].(FnDef)
	require.True(t, ok)
	assert.Equal(t, "formatNumber", fn.Name)
	assert.Len(t, fn.Params, 2)
}

func TestParseFnDefaultParamNumeric(t *testing.T) {
	// parquet.hexpat: fn idx_field_by_id(ref ThriftStruct s, s16 field_id, s16 since_idx = 0)
	input := `fn foo(s16 field_id, s16 since_idx = 0) {
    return field_id;
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	fn, ok := f.Items[0].(FnDef)
	require.True(t, ok)
	assert.Len(t, fn.Params, 2)
}

// =============================================================================
// Category C: Computed/assigned fields in struct/bitfield bodies
// Affects: vgm.hexpat, flc.hexpat, smk.hexpat
// =============================================================================

func TestParseBitfieldComputedField(t *testing.T) {
	// vgm.hexpat: versionValue = major * 100 + minor * 10 + bugfix; inside bitfield
	input := `bitfield VGMVersion {
    bugfix : 4;
    minor : 4;
    major : 24;
    versionValue = major * 100 + minor * 10 + bugfix;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseStructVarDeclWithInit(t *testing.T) {
	// flc.hexpat: u8 r8 = r << 2 | r >> 4; inside struct
	input := `struct Color {
    padding : 2;
    r : 6;
    u8 r8 = r << 2 | r >> 4;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseBitfieldVarDeclWithInit(t *testing.T) {
	// smk.hexpat: u32 size = dwordCount * 4; inside bitfield
	input := `bitfield FrameSize {
    keyframe : 1;
    padding : 1;
    dwordCount : 30;
    u32 size = dwordCount * 4;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category D: Double semicolons
// Affects: lua40.hexpat, pef.hexpat, pex.hexpat
// =============================================================================

func TestParseDoubleSemicolon(t *testing.T) {
	// lua40.hexpat: LuaFunction toplevelFunction @ sizeof(header);;
	input := `u32 value @ 0x00;;`
	f, err := Parse(input)
	require.NoError(t, err)
	require.NotEmpty(t, f.Items)
}

func TestParseBitfieldDoubleSemicolon(t *testing.T) {
	// pef.hexpat: relCount : 6;;
	input := `bitfield Foo {
    op : 2;
    relCount : 6;;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseStructDoubleSemicolonAfterAttrs(t *testing.T) {
	// pex.hexpat: } [[format("...")]];;
	input := `struct Foo {
    u8 x;
} [[format("bar")]];;`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category E: Comma-separated variable declarations
// Affects: ico.hexpat
// =============================================================================

func TestParseCommaSeparatedVarDecl(t *testing.T) {
	// ico.hexpat: u8 width, height;
	input := `struct Entry {
    u8 width, height;
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)
}

// =============================================================================
// Category F: `this` keyword in expressions
// Affects: binka.hexpat, and many files using [[name(std::format("...", this))]]
// =============================================================================

func TestParseThisKeyword(t *testing.T) {
	input := `struct Foo {
    u8 x [[name(std::format("Value: {}", this))]];
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category G: Anonymous/unnamed fields (type without name)
// Affects: binka.hexpat
// =============================================================================

func TestParseAnonymousField(t *testing.T) {
	// binka.hexpat: u8; (field with no name)
	input := `struct Foo {
    u8;
    u32;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category H: `padding : N` (colon form) inside struct bodies
// Affects: 3ds.hexpat, flc.hexpat (bitfield-style padding in struct)
// =============================================================================

func TestParsePaddingColonInStruct(t *testing.T) {
	// 3ds.hexpat: padding : 8; inside struct
	input := `struct Foo {
    u32 x;
    padding : 8;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category I: Template parameters on bitfields
// Affects: lznt1.hexpat
// =============================================================================

func TestParseTemplateBitfield(t *testing.T) {
	// lznt1.hexpat: bitfield CompressedTuple<auto lengthSize, auto displacementSize>
	input := `bitfield CompressedTuple<auto lengthSize, auto displacementSize> {
    length : lengthSize;
    displacement : displacementSize;
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	bf, ok := f.Items[0].(BitfieldDef)
	require.True(t, ok)
	assert.Equal(t, "CompressedTuple", bf.Name)
}

// =============================================================================
// Category J: Conditionals (if/else) inside bitfields
// Affects: flac.hexpat, adts.hexpat
// =============================================================================

func TestParseBitfieldConditional(t *testing.T) {
	// flac.hexpat: if (riceParameter == 0b1111) bitsPerSample : 5;
	input := `bitfield Rice {
    partitionOrder : 4;
    riceParameter : 4;
    if (riceParameter == 0b1111)
        bitsPerSample : 5;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseBitfieldConditionalBlock(t *testing.T) {
	// adts.hexpat: if (0 == Protection_absence) { u16 CRC16; }
	input := `bitfield Header {
    Protection_absence : 1;
    if (0 == Protection_absence) {
        CRC16 : 16;
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category K: Match inside bitfield
// Affects: hinf_luas.hexpat
// =============================================================================

func TestParseBitfieldMatch(t *testing.T) {
	// hinf_luas.hexpat: match(OpCode) { ... } inside bitfield
	input := `bitfield LuaBitfield {
    OpCode : 7;
    match(OpCode) {
        (0): args : 25;
        (_): other : 25;
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category L: Endian-qualified cast in enum values
// Affects: macho.hexpat
// =============================================================================

func TestParseEnumEndianCastValue(t *testing.T) {
	// macho.hexpat: I860 = be u32(15),
	input := `enum CpuType : u32 {
    X86 = 7,
    I860 = be u32(15),
    ARM = 12,
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)

	e, ok := f.Items[0].(EnumDef)
	require.True(t, ok)
	assert.Len(t, e.Members, 3)
}

// =============================================================================
// Category M: `auto` variables inside struct body
// Affects: stdfv4.hexpat
// =============================================================================

func TestParseAutoVarInStruct(t *testing.T) {
	// stdfv4.hexpat: auto start = $;
	input := `struct Foo {
    auto start = $;
    u32 value;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category N: Top-level expression statements (function calls)
// Affects: 7z.hexpat
// =============================================================================

func TestParseTopLevelFnCall(t *testing.T) {
	// 7z.hexpat: std::print("...");
	input := `std::print("hello");`
	f, err := Parse(input)
	require.NoError(t, err)
	require.NotEmpty(t, f.Items)
}

func TestParseTopLevelIfStmt(t *testing.T) {
	// 7z.hexpat: top-level if statement
	input := `u32 x @ 0;
if (x == 1) {
    u32 y @ 4;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category O: Match with range patterns in functions
// Affects: java_class.hexpat
// =============================================================================

func TestParseMatchRangeInFunction(t *testing.T) {
	// java_class.hexpat: match(frame_type) { (0 ... 63): return "SAME"; }
	input := `fn describe(u8 x) {
    match(x) {
        (0 ... 63): return "low";
        (64 ... 127): return "mid";
        (_): return "high";
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category P: `str` used as a function call / cast
// Affects: mp4.hexpat
// =============================================================================

func TestParseStrCast(t *testing.T) {
	// mp4.hexpat: match (str(type)) { ... }
	input := `struct Foo {
    u32 type;
    match (str(type)) {
        ("abc"): u32 a;
        (_): u32 b;
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category Q: Unsized char arrays (null-terminated strings)
// Affects: unity-asset-bundle.hexpat
// =============================================================================

func TestParseUnsizedCharArray(t *testing.T) {
	// unity-asset-bundle.hexpat: char signature[];
	input := `struct Header {
    char signature[];
    u32 version;
};`
	f, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, f.Items, 1)
}

// =============================================================================
// Category R: `in` keyword in variable placement
// Affects: blend.hexpat
// =============================================================================

func TestParseVarDeclWithIn(t *testing.T) {
	// blend.hexpat: type::RGBA8 image[size] @ 0x00 in thumbnailFlipped;
	input := `u8 data[10] @ 0x00 in someSection;`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category S: #ifdef wrapping attributes between } and ;
// Affects: blend.hexpat
// =============================================================================

func TestParseIfdefBetweenBraceAndSemicolon(t *testing.T) {
	// blend.hexpat: } \n #ifdef __IMHEX__ \n [[attr]] \n #endif \n ;
	input := `struct Foo {
    u32 x;
}
#ifdef __IMHEX__
[[hex::visualize("bitmap", x)]]
#endif
;`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category T: `block` keyword as type name
// Affects: xex.hexpat
// =============================================================================

func TestParseBlockKeyword(t *testing.T) {
	// xex.hexpat: block data_blocks[while(!std::mem::eof())] @ 2;
	input := `block data_blocks[4] @ 2;`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category U: Digit separators in number literals (tick marks)
// Affects: macho.hexpat enum values like 0x100'0000
// =============================================================================

func TestParseDigitSeparatorInEnum(t *testing.T) {
	// macho.hexpat: ARM64 = CpuType::ARM | 0x100'0000,
	input := `enum CpuType : u32 {
    ARM = 12,
    ARM64 = CpuType::ARM | 0x100'0000,
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

// =============================================================================
// Category V: Constructs that work in isolation (regression guards)
// These were once suspected as root causes for bulk file failures but are
// actually handled correctly. Kept as regression guards.
// =============================================================================

func TestParseStringComparisonWithEscape(t *testing.T) {
	input := `struct Foo {
    char signature[];
    if (signature == "UnityArchive\0") {
        u32 x;
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseBitfieldTypedEntry(t *testing.T) {
	input := `bitfield DateTime {
    s64 Ticks : 62;
    DateTimeKind kind : 2;
} [[bitfield_order(BitfieldOrder::LeastToMostSignificant, 64)]];`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseTripleAsteriskBlockComment(t *testing.T) {
	input := `/***
enum Foo : u8 {
    A = 1,
    B = 2,
};
***/
u32 x @ 0;`
	f, err := Parse(input)
	require.NoError(t, err)
	require.NotEmpty(t, f.Items)
}

func TestParseBreakInStructBody(t *testing.T) {
	input := `struct Header {
    u32 type;
    if (type == 5) {
        break;
    }
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseAnonymousEnum(t *testing.T) {
	input := `enum {
    A = 1,
    B = 2,
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseTemplateTypeVarDecl(t *testing.T) {
	input := `struct Foo {
    std::mem::Bytes<16> hash;
};`
	_, err := Parse(input)
	require.NoError(t, err)
}

func TestParseForLoopInsideIfDefNoTrailingSemicolon(t *testing.T) {
	input := `#ifdef __IMHEX__
    u128 previousSectionSize = 0;
    for (u32 i = 0, i < 10, i = i + 1) {
        std::assert(i < 20, "too big");
        previousSectionSize += i;
    }
#endif`
	_, err := Parse(input)
	assert.NoError(t, err)
}

// =============================================================================
// Minimal reproductions for the 8 remaining bulk file failures.
// Each test is the smallest snippet that triggers the actual parse bug.
// =============================================================================

// --- Root cause 1: CRLF line endings in block comments ---
// Affects: pickle.hexpat, flv.hexpat, dotnet_binaryformatter.hexpat
//
// The pcom-go library's Consume() calls ProgressLine() when it encounters \r,
// which advances s.Offset by 2 for a \r\n pair. But Consume's own loop variable
// "end" only advances by 1. This causes s.Offset to drift ahead of reality.
// The blockComment parser triggers this by calling Consume(end-start) where end
// was computed by scanning bytes directly (counting \r and \n as 1 byte each).
// Result: with enough CRLF lines, the offset overshoots and either panics or
// reads garbage, causing cascading parse failures.

func TestMinimalRepro_CRLFBlockComment(t *testing.T) {
	// Block comment with CRLF line endings — panics in pcom-go isCRLF
	input := "/*\r\ncomment\r\n*/\r\nu32 x;\r\n"
	defer func() {
		if r := recover(); r != nil {
			// Expected: pcom-go panics on CRLF in block comments
			return
		}
	}()
	_, err := Parse(input)
	// If we reach here without panic, the parse should have failed due to offset drift
	assert.Error(t, err, "CRLF in block comment causes offset drift in pcom-go Consume/ProgressLine — "+
		"affects pickle.hexpat, flv.hexpat, dotnet_binaryformatter.hexpat")
}

// --- Root cause 2: Struct inheritance with template args on parent type ---
// Affects: dotnet_binaryformatter.hexpat (in addition to CRLF above)
//
// struct StringValueWithCode: PrimitiveTypeEnumT<PrimitiveTypeEnum::String> { ... }
// The struct parser handles : Parent but not : Parent<TemplateArg>.

func TestMinimalRepro_StructInheritanceTemplateParent(t *testing.T) {
	input := `struct Foo: Bar<T> {
    u32 x;
};`
	_, err := Parse(input)
	assert.Error(t, err, "struct inheritance with template args on parent type — "+
		"affects dotnet_binaryformatter.hexpat line 299")
}

// --- Root cause 3: Bare identifier (unexpanded macro) followed by if keyword ---
// Affects: q3demo.hexpat
//
// q3demo.hexpat uses: ret = readbits(...);DECODERET
// After the semicolon, the parser sees DECODERET as the start of a new statement.
// It tries varDeclParser which treats DECODERET as a type name and consumes the
// following "if" keyword as a variable name, corrupting the parse state.

func TestMinimalRepro_BareIdentifierThenIf(t *testing.T) {
	input := `fn foo(){
    IDENT
    if(1 != 0){
        return 1;
    }
    return 0;
};`
	_, err := Parse(input)
	assert.Error(t, err, "bare identifier (unexpanded macro) before if keyword — "+
		"varDeclParser consumes 'if' as variable name, corrupting parse state — "+
		"affects q3demo.hexpat")
}

// --- Root cause 4: Anonymous type with [[inline]] attribute in match arm ---
// Affects: rar.hexpat, java_class.hexpat
//
// match (x) { (0): SomeType [[inline]]; }
// The match arm body parser sees SomeType as a type, then [[inline]] starts with [
// which it interprets as an array size bracket, not an attribute.

func TestMinimalRepro_MatchArmAnonymousTypeWithInlineAttr(t *testing.T) {
	input := `struct Foo {
    u32 x;
    match (x) {
        (0): Bar [[inline]];
    }
};`
	_, err := Parse(input)
	assert.Error(t, err, "anonymous type with [[inline]] in match arm — "+
		"[[ is misinterpreted as array bracket instead of attribute — "+
		"affects rar.hexpat, java_class.hexpat")
}

// --- Root cause 5: Stray semicolon after for-loop inside #ifdef ---
// Affects: blend.hexpat
//
// The for loop body closes with }, then a stray ; follows: };
// At file level, fileParser skips stray semicolons. But inside #ifdef,
// parseIfDef calls itemParser which does NOT skip stray semicolons.

func TestMinimalRepro_ForLoopSemicolonInsideIfDef(t *testing.T) {
	input := `#ifdef __IMHEX__
    for (u32 i = 0, i < 10, i = i + 1) {
        u32 x = i;
    };
#endif`
	_, err := Parse(input)
	assert.Error(t, err, "stray semicolon after for-loop inside #ifdef — "+
		"parseIfDef does not skip stray semicolons like fileParser does — "+
		"affects blend.hexpat")
}

// --- Root cause 6: Placement with integer section + attributes ---
// Affects: unity-asset-bundle.hexpat
//
// u8 data[size] @ cursor in 0 [[sealed]];
// The parser doesn't properly implement "@ expr in section" — it parses @ expr
// and treats "in" as a separate type name. When the section is an integer (0)
// rather than an identifier, the fallback VarDecl parse path fails because 0
// can't be a variable name, especially with [[attrs]] following.

func TestMinimalRepro_PlacementIntegerSectionWithAttrs(t *testing.T) {
	input := `u8 data[10] @ 0 in 0 [[sealed]];`
	_, err := Parse(input)
	assert.Error(t, err, "@ expr in <integer> [[attrs]] — 'in section' not properly parsed, "+
		"integer section with attributes fails — affects unity-asset-bundle.hexpat")
}

// =============================================================================
// Bulk file regression tests — one per currently-failing file
// These verify whole-file parsing once fixes are in place.
// =============================================================================

func TestParseBulkFile_3ds(t *testing.T) {
	input := readHexpatFile(t, "3ds.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_7z(t *testing.T) {
	input := readHexpatFile(t, "7z.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_adts(t *testing.T) {
	input := readHexpatFile(t, "adts.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_binka(t *testing.T) {
	input := readHexpatFile(t, "binka.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_blend(t *testing.T) {
	input := readHexpatFile(t, "blend.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_dos(t *testing.T) {
	input := readHexpatFile(t, "dos.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_dotnet_binaryformatter(t *testing.T) {
	input := readHexpatFile(t, "dotnet_binaryformatter.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_flac(t *testing.T) {
	input := readHexpatFile(t, "flac.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_flc(t *testing.T) {
	input := readHexpatFile(t, "flc.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_flv(t *testing.T) {
	input := readHexpatFile(t, "flv.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_hinf_luas(t *testing.T) {
	input := readHexpatFile(t, "hinf_luas.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_ico(t *testing.T) {
	input := readHexpatFile(t, "ico.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_ip(t *testing.T) {
	input := readHexpatFile(t, "ip.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_java_class(t *testing.T) {
	input := readHexpatFile(t, "java_class.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_kindle_update(t *testing.T) {
	input := readHexpatFile(t, "kindle_update.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_lua40(t *testing.T) {
	input := readHexpatFile(t, "lua40.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_lznt1(t *testing.T) {
	input := readHexpatFile(t, "lznt1.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_macho(t *testing.T) {
	input := readHexpatFile(t, "macho.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_mp4(t *testing.T) {
	input := readHexpatFile(t, "mp4.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_parquet(t *testing.T) {
	input := readHexpatFile(t, "parquet.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_pef(t *testing.T) {
	input := readHexpatFile(t, "pef.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_pex(t *testing.T) {
	input := readHexpatFile(t, "pex.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_pickle(t *testing.T) {
	input := readHexpatFile(t, "pickle.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_q3demo(t *testing.T) {
	input := readHexpatFile(t, "q3demo.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_rar(t *testing.T) {
	input := readHexpatFile(t, "rar.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_smk(t *testing.T) {
	input := readHexpatFile(t, "smk.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_stdfv4(t *testing.T) {
	input := readHexpatFile(t, "stdfv4.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_ttf(t *testing.T) {
	input := readHexpatFile(t, "ttf.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_unity_asset_bundle(t *testing.T) {
	input := readHexpatFile(t, "unity-asset-bundle.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_vgm(t *testing.T) {
	input := readHexpatFile(t, "vgm.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}

func TestParseBulkFile_xex(t *testing.T) {
	input := readHexpatFile(t, "xex.hexpat")
	_, err := Parse(input)
	assert.NoError(t, err)
}
