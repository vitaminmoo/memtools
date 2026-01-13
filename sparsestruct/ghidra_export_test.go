package sparsestruct_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

func TestGenerateCDefinitions(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Val uint32
	}

	type TestStruct struct {
		Field1 uint8
		Field2 uint8 `offset:"0x4"`
		// Pointer to nested struct
		Ptr *sparsestruct.PointerGetter[Inner]
		// Arrays
		Arr [4]uint16
		// String pointer
		Str *sparsestruct.StringPointer
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, TestStruct{})
	require.NoError(t, err)

	output := buf.String()
	
	// Check for includes
	assert.Contains(t, output, "#include <stdint.h>")
	
	// Check for TestStruct definition
	assert.Contains(t, output, "typedef struct TestStruct TestStruct;")
	assert.Contains(t, output, "struct TestStruct {")
	
	// Check fields
	assert.Contains(t, output, "uint8_t Field1;")
	// Padding: Field1 is size 1. Offset becomes 1. Field2 is at 4.
	// Padding 3 bytes in Ghidra-style format
	assert.Contains(t, output, "uint8_t undefined_0x1[3];")
	assert.Contains(t, output, "uint8_t Field2;")
	
	// Pointer - uses "struct Inner *" to avoid field/type name shadowing
	assert.Contains(t, output, "typedef struct Inner Inner;")
	assert.Contains(t, output, "struct Inner * Ptr;")
	
	// Array: [4]uint16 -> uint16_t Arr[4]
	assert.Contains(t, output, "uint16_t Arr[4];")
	
	// StringPointer -> char *
	assert.Contains(t, output, "char * Str;")
	
	// Check Inner struct definition presence
	assert.Contains(t, output, "struct Inner {")
	assert.Contains(t, output, "uint32_t Val;")
}

func TestGenerateCDefinitions_Anon(t *testing.T) {
	t.Parallel()
	
	v := struct {
		A uint64
	}{}
	
	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, v)
	require.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "typedef struct Struct_1 Struct_1;")
	assert.Contains(t, output, "struct Struct_1 {")
	assert.Contains(t, output, "uint64_t A;")
}

func TestGenerateCDefinitions_ComplexArray(t *testing.T) {
	t.Parallel()

	type ArrayStruct struct {
		Matrix [3][4]uint8
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, ArrayStruct{})
	require.NoError(t, err)

	output := buf.String()
	// [3][4]uint8 -> uint8_t Matrix[3][4];
	assert.Contains(t, output, "uint8_t Matrix[3][4];")
}

func TestGenerateCDefinitions_Arch32(t *testing.T) {
	t.Parallel()

	// Test struct where pointer size matters for offset calculation
	// Ptr at offset 0 (size 4 on 32-bit), Field after at offset 4
	type PtrStruct struct {
		Ptr   *sparsestruct.StringPointer
		Field uint32 `offset:"0x4"`
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitionsWithArch(&buf, sparsestruct.Arch32, PtrStruct{})
	require.NoError(t, err)

	output := buf.String()
	// Pointer should be char * (4 bytes on 32-bit)
	assert.Contains(t, output, "char * Ptr;")
	// Field at 0x4 should follow immediately after 4-byte pointer - no padding needed
	assert.NotContains(t, output, "undefined_0x4")
	assert.Contains(t, output, "uint32_t Field;")
}

func TestGenerateCDefinitions_Arch64(t *testing.T) {
	t.Parallel()

	// Same struct but 64-bit - pointer is 8 bytes, so Field at 0x4 needs padding
	type PtrStruct struct {
		Ptr   *sparsestruct.StringPointer
		Field uint32 `offset:"0x4"`
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitionsWithArch(&buf, sparsestruct.Arch64, PtrStruct{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)
	// 64-bit pointer (8 bytes) at offset 0
	// Field at 0x4 means we actually go backwards - this is a broken struct definition
	// but the code should handle it by not adding negative padding
	assert.Contains(t, output, "char * Ptr;")
}

// Package-level types for cyclic reference testing
type testNodeA struct {
	Value uint32
	Child *sparsestruct.PointerGetter[testNodeB] `offset:"0x8"`
}

type testNodeB struct {
	Value  uint32
	Parent *sparsestruct.PointerGetter[testNodeA] `offset:"0x8"`
}

type testListNode struct {
	Value uint64
	Next  *sparsestruct.PointerGetter[testListNode] `offset:"0x8"`
	Prev  *sparsestruct.PointerGetter[testListNode] `offset:"0x10"`
}

func TestGenerateCDefinitions_CyclicTypes(t *testing.T) {
	t.Parallel()

	// Test that cyclic type references work correctly
	// A -> B -> A (cycle)
	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, testNodeA{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// Both types should be forward-declared
	assert.Contains(t, output, "typedef struct testNodeA testNodeA;")
	assert.Contains(t, output, "typedef struct testNodeB testNodeB;")

	// Both structs should be defined
	assert.Contains(t, output, "struct testNodeA {")
	assert.Contains(t, output, "struct testNodeB {")

	// Pointers should reference each other
	assert.Contains(t, output, "struct testNodeB * Child;")
	assert.Contains(t, output, "struct testNodeA * Parent;")
}

func TestGenerateCDefinitions_SelfReferential(t *testing.T) {
	t.Parallel()

	// Test self-referential type (linked list node)
	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, testListNode{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	assert.Contains(t, output, "typedef struct testListNode testListNode;")
	assert.Contains(t, output, "struct testListNode {")
	assert.Contains(t, output, "struct testListNode * Next;")
	assert.Contains(t, output, "struct testListNode * Prev;")
}

func TestGenerateCDefinitions_StringWithMaxlen(t *testing.T) {
	t.Parallel()

	type StringStruct struct {
		Name  string `offset:"0x0,maxlen:32"`
		Desc  string `offset:"0x20,maxlen:64"`
		Count uint32 `offset:"0x60"`
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, StringStruct{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// String with maxlen should become char[N]
	assert.Contains(t, output, "char Name[32];")
	assert.Contains(t, output, "char Desc[64];")
	assert.Contains(t, output, "uint32_t Count;")
}

func TestGenerateCDefinitions_StringWithoutMaxlen(t *testing.T) {
	t.Parallel()

	type BadStruct struct {
		Name string `offset:"0x0"`
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, BadStruct{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxlen")
}

// Package-level types for embedded struct testing
type embeddedBase struct {
	A uint32 `offset:"0x00"`
	B uint32 `offset:"0x08"`
}

type embeddedExtended struct {
	embeddedBase
	C uint32 `offset:"0x04"` // fills gap between A and B
	D uint32 `offset:"0x0C"` // after B
}

func TestGenerateCDefinitions_Embedded(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, embeddedExtended{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// Should have typedef and struct definition for Extended only (Base is inlined)
	assert.Contains(t, output, "typedef struct embeddedExtended embeddedExtended;")
	assert.Contains(t, output, "struct embeddedExtended {")

	// Should NOT have Base as a separate type since it's inlined
	assert.NotContains(t, output, "typedef struct embeddedBase")

	// Fields should be in offset order: A (0x00), C (0x04), B (0x08), D (0x0C)
	assert.Contains(t, output, "uint32_t A;")
	assert.Contains(t, output, "uint32_t C;")
	assert.Contains(t, output, "uint32_t B;")
	assert.Contains(t, output, "uint32_t D;")

	// Verify the ordering by checking A comes before C in the output
	aPos := strings.Index(output, "uint32_t A;")
	cPos := strings.Index(output, "uint32_t C;")
	bPos := strings.Index(output, "uint32_t B;")
	dPos := strings.Index(output, "uint32_t D;")

	assert.True(t, aPos < cPos, "A should come before C")
	assert.True(t, cPos < bPos, "C should come before B")
	assert.True(t, bPos < dPos, "B should come before D")
}

// Test embedded struct with pointer fields
type embeddedWithPointerBase struct {
	ID uint32 `offset:"0x00"`
}

type embeddedWithPointerExtended struct {
	embeddedWithPointerBase
	Data *sparsestruct.PointerGetter[embeddedWithPointerBase] `offset:"0x08"`
}

func TestGenerateCDefinitions_EmbeddedWithPointer(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, embeddedWithPointerExtended{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// Extended struct should have inlined Base fields plus the pointer
	assert.Contains(t, output, "struct embeddedWithPointerExtended {")
	assert.Contains(t, output, "uint32_t ID;")
	assert.Contains(t, output, "struct embeddedWithPointerBase * Data;")

	// Base struct should be generated separately because it's referenced by pointer
	assert.Contains(t, output, "typedef struct embeddedWithPointerBase embeddedWithPointerBase;")
	assert.Contains(t, output, "struct embeddedWithPointerBase {")
}

// Nested embedding test types
type nestedLevel2 struct {
	X uint32 `offset:"0x00"`
}

type nestedLevel1 struct {
	nestedLevel2
	Y uint32 `offset:"0x04"`
}

type nestedTop struct {
	nestedLevel1
	Z uint32 `offset:"0x08"`
}

func TestGenerateCDefinitions_NestedEmbedding(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, nestedTop{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// Only nestedTop should be a typedef (Level1 and Level2 are inlined)
	assert.Contains(t, output, "typedef struct nestedTop nestedTop;")
	assert.NotContains(t, output, "typedef struct nestedLevel1")
	assert.NotContains(t, output, "typedef struct nestedLevel2")

	// All fields should be present in offset order
	assert.Contains(t, output, "uint32_t X;")
	assert.Contains(t, output, "uint32_t Y;")
	assert.Contains(t, output, "uint32_t Z;")

	xPos := strings.Index(output, "uint32_t X;")
	yPos := strings.Index(output, "uint32_t Y;")
	zPos := strings.Index(output, "uint32_t Z;")

	assert.True(t, xPos < yPos, "X should come before Y")
	assert.True(t, yPos < zPos, "Y should come before Z")
}

func TestGenerateCDefinitions_UintptrAsVoidPointer(t *testing.T) {
	t.Parallel()

	// uintptr should map to void* for generic/variant pointers
	type BaseWithVariant struct {
		Type       uint32  `offset:"0x00"`
		DataOrName uintptr `offset:"0x08"` // generic pointer - could be different types
		CommonPtr  uintptr `offset:"0x10"` // another generic pointer
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitions(&buf, BaseWithVariant{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// uintptr fields should become void*
	assert.Contains(t, output, "void * DataOrName;")
	assert.Contains(t, output, "void * CommonPtr;")
}

func TestGenerateCDefinitions_UintptrArch32(t *testing.T) {
	t.Parallel()

	// Test that uintptr uses correct size for 32-bit arch
	type PtrStruct struct {
		Ptr   uintptr `offset:"0x00"`
		Field uint32  `offset:"0x04"` // right after 4-byte pointer on 32-bit
	}

	var buf bytes.Buffer
	err := sparsestruct.GenerateCDefinitionsWithArch(&buf, sparsestruct.Arch32, PtrStruct{})
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	// Should have void* followed immediately by Field (no padding)
	assert.Contains(t, output, "void * Ptr;")
	assert.Contains(t, output, "uint32_t Field;")
	assert.NotContains(t, output, "undefined_0x4") // no padding needed on 32-bit
}
