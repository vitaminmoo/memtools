package process_test

import (
	"os"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitaminmoo/memtools/hexpat/runtime"
	"github.com/vitaminmoo/memtools/process"
)

func TestReadBatchSelf(t *testing.T) {
	// process_vm_readv supports reading from your own pid; use that to verify
	// every requested iovec gets the expected bytes.
	src1 := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	src2 := []byte{0x01, 0x02, 0x03}
	src3 := []byte{0xCA, 0xFE}

	p := process.New(os.Getpid())
	reqs := []runtime.ReadReq{
		{Addr: uintptr(unsafe.Pointer(&src1[0])), Buf: make([]byte, len(src1))},
		{Addr: uintptr(unsafe.Pointer(&src2[0])), Buf: make([]byte, len(src2))},
		{Addr: uintptr(unsafe.Pointer(&src3[0])), Buf: make([]byte, len(src3))},
	}
	require.NoError(t, p.ReadBatch(reqs))
	assert.Equal(t, src1, reqs[0].Buf)
	assert.Equal(t, src2, reqs[1].Buf)
	assert.Equal(t, src3, reqs[2].Buf)
}

func TestReadBatchEmpty(t *testing.T) {
	p := process.New(os.Getpid())
	assert.NoError(t, p.ReadBatch(nil))
	assert.NoError(t, p.ReadBatch([]runtime.ReadReq{}))
}

// TestReadBatchManyChunks confirms the chunking loop runs more than once when
// the request count exceeds maxBatchIov (1024).
func TestReadBatchManyChunks(t *testing.T) {
	const n = 1500
	src := make([]byte, n)
	for i := range src {
		src[i] = byte(i % 256)
	}

	p := process.New(os.Getpid())
	reqs := make([]runtime.ReadReq, n)
	for i := range reqs {
		reqs[i] = runtime.ReadReq{
			Addr: uintptr(unsafe.Pointer(&src[i])),
			Buf:  make([]byte, 1),
		}
	}
	require.NoError(t, p.ReadBatch(reqs))
	for i := range reqs {
		assert.Equal(t, src[i], reqs[i].Buf[0], "req %d", i)
	}
}
