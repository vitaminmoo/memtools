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
	One   uin8   `offset:"0x0,le"`
	Three uint32 `offset:"0x1,le"`
}

type TestTop struct {
	First  uint32 `offset:"0x0,le"`
	Second uint32 `offset:"0x4,le"`
	Fourth sparsestruct.PointerGetter[*TestSecond]
}

func TestRead(t *testing.T) {
	process := ptrchain.New(os.Getpid())

	var eh struct{}
	err := process.Read(t.Context(), 0x400000, &eh)
	require.NoError(t, err)
	assert.Equal(t, uint32(0x7f454c46), eh.Mag)
	assert.Equal(t, uint8(0x02), eh.Class)
	assert.Equal(t, uint8(0x01), eh.Data)
	assert.Equal(t, uint64(0x400008), eh.Entry.Address())
}
