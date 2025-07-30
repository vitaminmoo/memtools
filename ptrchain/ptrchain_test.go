package ptrchain_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaminmoo/memtools/ptrchain"
	"github.com/vitaminmoo/memtools/sparsestruct"
)

func TestRead(t *testing.T) {
	/*
		maps, err := pidmaps.Maps(os.Getpid())
		require.NoError(t, err)
		for _, m := range maps {
			t.Logf("Map: %#v\n", m)
		}
	*/

	process := ptrchain.New(os.Getpid())
	var eh struct {
		Mag   uint32 `offset:"0x0,be"`
		Class uint8
		Data  uint8
		Entry sparsestruct.PointerGetter `offset:"0x18"`
	}
	err := process.Read(t.Context(), 0x400000, &eh)
	require.NoError(t, err)
	assert.Equal(t, uint32(0x7f454c46), eh.Mag)
	assert.Equal(t, uint8(0x02), eh.Class)
	assert.Equal(t, uint8(0x01), eh.Data)
	assert.Equal(t, uint64(0x400008), eh.Entry.Address())

	var entry struct {
		one uint64 `offset:"0x0,le"`
		two uint64 `offset:"0x8,le"`
	}
}
