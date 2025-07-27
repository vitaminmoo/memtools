package sparsestruct_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	var v struct {
		Field1 uint8 `offset:"1"`
		Field2 uint8 `offset:"0b10"`
		Field3 uint8
		Field4 uint8
		// skip one
		Field6 uint8 `offset:"0x06"`
	}

	err := sparsestruct.Unmarshal(data, &v)
	require.NoError(t, err)

	assert.Equal(t, uint8(1), v.Field1, "Field1 should be 1")
	assert.Equal(t, uint8(2), v.Field2, "Field2 should be 2")
	assert.Equal(t, uint8(3), v.Field3, "Field3 should be 3")
	assert.Equal(t, uint8(4), v.Field4, "Field4 should be 4")
	assert.Equal(t, uint8(6), v.Field6, "Field6 should be 6")
}

func TestPointer(t *testing.T) {
	t.Parallel()
	data := []byte{0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	var v struct {
		Field1 sparsestruct.PointerGetter `offset:"0"`
	}

	err := sparsestruct.Unmarshal(data, &v)
	require.NoError(t, err)

	data2 := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	var v2 struct {
		Field1 uint64 `offset:"0"`
		Field2 uint64 `offset:"8"`
	}

	assert.Equal(t, v.Field1.Address(), uint64(0xF8), "Field1 should be at address 0xF8")

	needed, err := v.Field1.Length(&v2)
	require.NoError(t, err)
	assert.Equal(t, uint64(16), needed, "v2 should only need 16 bytes")

	err = v.Field1.Get(0xF0, data2, &v2)
	require.NoError(t, err)

	assert.Equal(t, uint64(1), v2.Field1, "Field1 should be 1")
	assert.Equal(t, uint64(2), v2.Field2, "Field2 should be 2")
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := sparsestruct.Unmarshal(tc.data, tc.v)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.v)
		})
	}
}
