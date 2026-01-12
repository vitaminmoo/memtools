package sparsestruct_test

import (
	"bytes"
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
