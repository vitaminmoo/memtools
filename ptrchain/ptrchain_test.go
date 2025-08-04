package ptrchain_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/ptrchain"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

type TestSecond struct {
	One   uint8  `offset:"0x0,le"`
	Three uint32 `offset:"0x1,le"`
}

type TestTop struct {
	First  uint32 `offset:"0x0,be"`
	Second uint8  `offset:"0x4,le"`
	Fourth *sparsestruct.PointerGetter[TestSecond]
}

func TestRead(t *testing.T) {
	process := ptrchain.New(os.Getpid())

	var eh TestTop
	err := process.Read(t.Context(), 0x400000, &eh)
	require.NoError(t, err)
	assert.Equal(t, uint32(0x7f454c46), eh.First)
	assert.Equal(t, uint8(0x02), eh.Second)
	assert.Equal(t, uint64(0x400008), eh.Fourth.Address())
}
