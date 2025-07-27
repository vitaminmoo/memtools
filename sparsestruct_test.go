package sparsestruct_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitaminmoo/sparsestruct"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	data := []byte{0x00, 0x01, 0x02, 0x03}
	var v struct {
		Field1 uint8 `offset:"1"`
		Field2 uint8 `offset:"2"`
	}

	err := sparsestruct.Unmarshal(data, &v)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assert.Equal(t, uint8(1), v.Field1, "Field1 should be 1")
	assert.Equal(t, uint8(2), v.Field2, "Field2 should be 2")
}

func TestPointer(t *testing.T) {
	t.Parallel()
	data := []byte{0x01, 0x02, 0x03, 0x04}
	var v struct {
		Field1 func() uintptr `offset:"0,pointer"`
	}

	err := sparsestruct.Unmarshal(data, &v)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assert.Equal(t, uintptr(0x01020304), v.Field1(), "Field1 should be 0x01020304")
}
