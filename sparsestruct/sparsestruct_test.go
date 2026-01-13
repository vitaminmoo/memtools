package sparsestruct_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	data := []byte{0xFF, 0xFF, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	r := bytes.NewReader(data)

	var v struct {
		Field0 uint8
		// skip one
		Field1 uint8 `offset:"0x02"`
		Field2 uint8 `offset:"0x03"`
		Field3 uint8
		Field4 uint8
		// skip one
		Field6 uint8 `offset:"0x07"`
	}

	err := sparsestruct.Unmarshal(r, 0, &v)
	require.NoError(t, err)

	assert.Equal(t, uint8(0xFF), v.Field0)
	assert.Equal(t, uint8(0x01), v.Field1)
	assert.Equal(t, uint8(0x02), v.Field2)
	assert.Equal(t, uint8(0x03), v.Field3)
	assert.Equal(t, uint8(0x04), v.Field4)
	assert.Equal(t, uint8(0x06), v.Field6)
}

func TestPointer(t *testing.T) {
	t.Parallel()
	data := []byte{
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	}
	r := bytes.NewReader(data)
	type DestStruct struct {
		Field1 uint8
		Field2 uint8 `offset:"4"`
	}
	type SourceStruct struct {
		DestPointer *sparsestruct.PointerGetter[DestStruct]
	}

	v := SourceStruct{}

	err := sparsestruct.Unmarshal(r, 0, &v)
	require.NoError(t, err)
	require.NotNil(t, v.DestPointer)

	assert.Equal(t, uintptr(0x10), v.DestPointer.Address())

	require.Equal(t, uint8(0x00), v.DestPointer.Value().Field1)
	require.Equal(t, uint8(0x04), v.DestPointer.Value().Field2)
}

func TestIntegerTypes(t *testing.T) {
	t.Parallel()

	uintMin := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	uintMax := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	intMinBE := []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	intMaxBE := []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	int16MinLE := []byte{0x00, 0x80}
	int32MinLE := []byte{0x00, 0x00, 0x00, 0x80}
	int64MinLE := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}

	int16MaxLE := []byte{0xff, 0x7f}
	int32MaxLE := []byte{0xff, 0xff, 0xff, 0x7f}
	int64MaxLE := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}

	testCases := []struct {
		name     string
		data     []byte
		v        any
		expected any
	}{
		// int8
		{
			name: "int8 min",
			data: intMinBE,
			v: &struct {
				V int8 `offset:"0"`
			}{},
			expected: &struct {
				V int8 `offset:"0"`
			}{V: math.MinInt8},
		},
		{
			name: "int8 max",
			data: intMaxBE,
			v: &struct {
				V int8 `offset:"0"`
			}{},
			expected: &struct {
				V int8 `offset:"0"`
			}{V: math.MaxInt8},
		},
		// uint8
		{
			name: "uint8 min",
			data: uintMin,
			v: &struct {
				V uint8 `offset:"0"`
			}{},
			expected: &struct {
				V uint8 `offset:"0"`
			}{V: 0},
		},
		{
			name: "uint8 max",
			data: uintMax,
			v: &struct {
				V uint8 `offset:"0"`
			}{},
			expected: &struct {
				V uint8 `offset:"0"`
			}{V: math.MaxUint8},
		},

		// int16
		{
			name: "int16 min le",
			data: int16MinLE,
			v: &struct {
				V int16 `offset:"0,le"`
			}{},
			expected: &struct {
				V int16 `offset:"0,le"`
			}{V: math.MinInt16},
		},
		{
			name: "int16 max le",
			data: int16MaxLE,
			v: &struct {
				V int16 `offset:"0,le"`
			}{},
			expected: &struct {
				V int16 `offset:"0,le"`
			}{V: math.MaxInt16},
		},
		{
			name: "int16 min be",
			data: intMinBE,
			v: &struct {
				V int16 `offset:"0,be"`
			}{},
			expected: &struct {
				V int16 `offset:"0,be"`
			}{V: math.MinInt16},
		},
		{
			name: "int16 max be",
			data: intMaxBE,
			v: &struct {
				V int16 `offset:"0,be"`
			}{},
			expected: &struct {
				V int16 `offset:"0,be"`
			}{V: math.MaxInt16},
		},

		// uint16
		{
			name: "uint16 min le",
			data: []byte{0, 0},
			v: &struct {
				V uint16 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint16 `offset:"0,le"`
			}{V: 0},
		},
		{
			name: "uint16 max le",
			data: uintMax,
			v: &struct {
				V uint16 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint16 `offset:"0,le"`
			}{V: math.MaxUint16},
		},
		{
			name: "uint16 min be",
			data: uintMin,
			v: &struct {
				V uint16 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint16 `offset:"0,be"`
			}{V: 0},
		},
		{
			name: "uint16 max be",
			data: uintMax,
			v: &struct {
				V uint16 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint16 `offset:"0,be"`
			}{V: math.MaxUint16},
		},

		// int32
		{
			name: "int32 min le",
			data: int32MinLE,
			v: &struct {
				V int32 `offset:"0,le"`
			}{},
			expected: &struct {
				V int32 `offset:"0,le"`
			}{V: math.MinInt32},
		},
		{
			name: "int32 max le",
			data: int32MaxLE,
			v: &struct {
				V int32 `offset:"0,le"`
			}{},
			expected: &struct {
				V int32 `offset:"0,le"`
			}{V: math.MaxInt32},
		},
		{
			name: "int32 min be",
			data: intMinBE,
			v: &struct {
				V int32 `offset:"0,be"`
			}{},
			expected: &struct {
				V int32 `offset:"0,be"`
			}{V: math.MinInt32},
		},
		{
			name: "int32 max be",
			data: intMaxBE,
			v: &struct {
				V int32 `offset:"0,be"`
			}{},
			expected: &struct {
				V int32 `offset:"0,be"`
			}{V: math.MaxInt32},
		},

		// uint32
		{
			name: "uint32 min le",
			data: []byte{0, 0, 0, 0},
			v: &struct {
				V uint32 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint32 `offset:"0,le"`
			}{V: 0},
		},
		{
			name: "uint32 max le",
			data: uintMax,
			v: &struct {
				V uint32 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint32 `offset:"0,le"`
			}{V: math.MaxUint32},
		},
		{
			name: "uint32 min be",
			data: uintMin,
			v: &struct {
				V uint32 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint32 `offset:"0,be"`
			}{V: 0},
		},
		{
			name: "uint32 max be",
			data: uintMax,
			v: &struct {
				V uint32 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint32 `offset:"0,be"`
			}{V: math.MaxUint32},
		},

		// int64
		{
			name: "int64 min le",
			data: int64MinLE,
			v: &struct {
				V int64 `offset:"0,le"`
			}{},
			expected: &struct {
				V int64 `offset:"0,le"`
			}{V: math.MinInt64},
		},
		{
			name: "int64 max le",
			data: int64MaxLE,
			v: &struct {
				V int64 `offset:"0,le"`
			}{},
			expected: &struct {
				V int64 `offset:"0,le"`
			}{V: math.MaxInt64},
		},
		{
			name: "int64 min be",
			data: intMinBE,
			v: &struct {
				V int64 `offset:"0,be"`
			}{},
			expected: &struct {
				V int64 `offset:"0,be"`
			}{V: math.MinInt64},
		},
		{
			name: "int64 max be",
			data: intMaxBE,
			v: &struct {
				V int64 `offset:"0,be"`
			}{},
			expected: &struct {
				V int64 `offset:"0,be"`
			}{V: math.MaxInt64},
		},

		// uint64
		{
			name: "uint64 min le",
			data: uintMin,
			v: &struct {
				V uint64 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint64 `offset:"0,le"`
			}{V: 0},
		},
		{
			name: "uint64 max le",
			data: uintMax,
			v: &struct {
				V uint64 `offset:"0,le"`
			}{},
			expected: &struct {
				V uint64 `offset:"0,le"`
			}{V: math.MaxUint64},
		},
		{
			name: "uint64 min be",
			data: uintMin,
			v: &struct {
				V uint64 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint64 `offset:"0,be"`
			}{V: 0},
		},
		{
			name: "uint64 max be",
			data: uintMax,
			v: &struct {
				V uint64 `offset:"0,be"`
			}{},
			expected: &struct {
				V uint64 `offset:"0,be"`
			}{V: math.MaxUint64},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bytes.NewReader(tc.data)
			err := sparsestruct.Unmarshal(r, 0, tc.v)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.v)
		})
	}
}

func TestArrayTypes(t *testing.T) {
	t.Parallel()

	t.Run("uint32 array little endian", func(t *testing.T) {
		t.Parallel()
		data := []byte{
			0x01, 0x00, 0x00, 0x00, // 1
			0x02, 0x00, 0x00, 0x00, // 2
			0x03, 0x00, 0x00, 0x00, // 3
			0x04, 0x00, 0x00, 0x00, // 4
			0x05, 0x00, 0x00, 0x00, // 5
			0x06, 0x00, 0x00, 0x00, // 6
			0x07, 0x00, 0x00, 0x00, // 7
			0x08, 0x00, 0x00, 0x00, // 8
		}
		r := bytes.NewReader(data)

		var v struct {
			Vals [8]uint32 `offset:"0,le"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		expected := [8]uint32{1, 2, 3, 4, 5, 6, 7, 8}
		assert.Equal(t, expected, v.Vals)
	})

	t.Run("int16 array big endian", func(t *testing.T) {
		t.Parallel()
		data := []byte{
			0x00, 0x01, // 1
			0x00, 0x02, // 2
			0x80, 0x00, // -32768 (MinInt16)
			0x7F, 0xFF, // 32767 (MaxInt16)
		}
		r := bytes.NewReader(data)

		var v struct {
			Vals [4]int16 `offset:"0,be"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		expected := [4]int16{1, 2, math.MinInt16, math.MaxInt16}
		assert.Equal(t, expected, v.Vals)
	})
}

func TestStringTypes(t *testing.T) {
	t.Parallel()

	t.Run("inline string with maxlen", func(t *testing.T) {
		t.Parallel()
		// "Hello" followed by null, then garbage, then next field
		data := []byte{
			'H', 'e', 'l', 'l', 'o', 0x00, 'X', 'X', // 8 bytes for Name
			0x42, 0x00, 0x00, 0x00, // Field = 0x42
		}
		r := bytes.NewReader(data)

		var v struct {
			Name  string `offset:"0x0,maxlen:8"`
			Field uint32 `offset:"0x8,le"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, "Hello", v.Name)
		assert.Equal(t, uint32(0x42), v.Field)
	})

	t.Run("inline string fills maxlen", func(t *testing.T) {
		t.Parallel()
		// String that fills the entire maxlen (no null terminator within bounds)
		data := []byte{
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', // 8 bytes, no null
			0x99, 0x00, 0x00, 0x00, // Field = 0x99
		}
		r := bytes.NewReader(data)

		var v struct {
			Name  string `offset:"0x0,maxlen:8"`
			Field uint32 `offset:"0x8,le"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, "ABCDEFGH", v.Name)
		assert.Equal(t, uint32(0x99), v.Field)
	})

	t.Run("empty string with maxlen", func(t *testing.T) {
		t.Parallel()
		// Empty string (starts with null)
		data := []byte{
			0x00, 'X', 'X', 'X', 'X', 'X', 'X', 'X', // 8 bytes, null at start
			0x42, 0x00, 0x00, 0x00, // Field = 0x42
		}
		r := bytes.NewReader(data)

		var v struct {
			Name  string `offset:"0x0,maxlen:8"`
			Field uint32 `offset:"0x8,le"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, "", v.Name)
		assert.Equal(t, uint32(0x42), v.Field)
	})

	t.Run("multiple inline strings", func(t *testing.T) {
		t.Parallel()
		data := []byte{
			'J', 'o', 'h', 'n', 0x00, 0x00, 0x00, 0x00, // FirstName (8 bytes)
			'D', 'o', 'e', 0x00, 0x00, 0x00, 0x00, 0x00, // LastName (8 bytes)
			0x1E, 0x00, 0x00, 0x00, // Age = 30
		}
		r := bytes.NewReader(data)

		var v struct {
			FirstName string `offset:"0x0,maxlen:8"`
			LastName  string `offset:"0x8,maxlen:8"`
			Age       uint32 `offset:"0x10,le"`
		}

		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, "John", v.FirstName)
		assert.Equal(t, "Doe", v.LastName)
		assert.Equal(t, uint32(30), v.Age)
	})
}

func TestEmbeddedStruct(t *testing.T) {
	t.Parallel()

	// Test basic embedding where extended struct fills gaps in base
	t.Run("basic embedding with gap filling", func(t *testing.T) {
		t.Parallel()

		type Base struct {
			A uint32 `offset:"0x00,le"`
			B uint32 `offset:"0x08,le"`
		}

		type Extended struct {
			Base
			C uint32 `offset:"0x04,le"` // fills gap between A and B
			D uint32 `offset:"0x0C,le"` // after B
		}

		data := []byte{
			0x01, 0x00, 0x00, 0x00, // A at 0x00 = 1
			0x03, 0x00, 0x00, 0x00, // C at 0x04 = 3
			0x02, 0x00, 0x00, 0x00, // B at 0x08 = 2
			0x04, 0x00, 0x00, 0x00, // D at 0x0C = 4
		}
		r := bytes.NewReader(data)

		var v Extended
		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, uint32(1), v.A) // from Base
		assert.Equal(t, uint32(2), v.B) // from Base
		assert.Equal(t, uint32(3), v.C) // from Extended
		assert.Equal(t, uint32(4), v.D) // from Extended
	})

	// Test nested embedding (A embeds B embeds C)
	t.Run("nested embedding", func(t *testing.T) {
		t.Parallel()

		type Level2 struct {
			X uint32 `offset:"0x00,le"`
		}

		type Level1 struct {
			Level2
			Y uint32 `offset:"0x04,le"`
		}

		type Top struct {
			Level1
			Z uint32 `offset:"0x08,le"`
		}

		data := []byte{
			0x0A, 0x00, 0x00, 0x00, // X at 0x00 = 10
			0x0B, 0x00, 0x00, 0x00, // Y at 0x04 = 11
			0x0C, 0x00, 0x00, 0x00, // Z at 0x08 = 12
		}
		r := bytes.NewReader(data)

		var v Top
		err := sparsestruct.Unmarshal(r, 0, &v)
		require.NoError(t, err)

		assert.Equal(t, uint32(10), v.X)
		assert.Equal(t, uint32(11), v.Y)
		assert.Equal(t, uint32(12), v.Z)
	})

	// Test Size() with embedded struct
	t.Run("size calculation", func(t *testing.T) {
		t.Parallel()

		type Base struct {
			A uint32 `offset:"0x00"`
			B uint32 `offset:"0x10"` // gap at 0x04-0x0F
		}

		type Extended struct {
			Base
			C uint32 `offset:"0x08"` // in the gap
		}

		// Size should be 0x10 + 4 = 0x14 = 20 (from Base.B)
		size, err := sparsestruct.Size(Extended{})
		require.NoError(t, err)
		assert.Equal(t, 20, size)
	})
}