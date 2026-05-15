package runtime

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAt(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	ctx := NewReadContext(bytes.NewReader(data))

	buf := make([]byte, 2)
	n, err := ctx.ReadAt(buf, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte{0x02, 0x03}, buf)
}

func TestReadAtSeeksCorrectly(t *testing.T) {
	data := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	ctx := NewReadContext(bytes.NewReader(data))

	// Read from end
	buf := make([]byte, 1)
	_, err := ctx.ReadAt(buf, 3)
	require.NoError(t, err)
	assert.Equal(t, byte(0xDD), buf[0])

	// Read from start (seeks back)
	_, err = ctx.ReadAt(buf, 0)
	require.NoError(t, err)
	assert.Equal(t, byte(0xAA), buf[0])
}

func TestReadAtError(t *testing.T) {
	data := []byte{0x01}
	ctx := NewReadContext(bytes.NewReader(data))

	buf := make([]byte, 4)
	_, err := ctx.ReadAt(buf, 0)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestVisit(t *testing.T) {
	ctx := NewReadContext(bytes.NewReader(nil))

	assert.False(t, ctx.Visit(0x1000))
	assert.True(t, ctx.Visit(0x1000))
	assert.False(t, ctx.Visit(0x2000))
	assert.True(t, ctx.Visit(0x2000))
}

func TestChainError(t *testing.T) {
	inner := io.ErrUnexpectedEOF
	ce := ChainError{Path: "Header.Magic", Address: 0x100, Err: inner}

	assert.Contains(t, ce.Error(), "Header.Magic")
	assert.Contains(t, ce.Error(), "0x100")
	assert.ErrorIs(t, ce, inner)
}

func TestReadBatchFallback(t *testing.T) {
	data := []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17}
	ctx := NewReadContext(bytes.NewReader(data))

	reqs := []ReadReq{
		{Addr: 0, Buf: make([]byte, 2)},
		{Addr: 5, Buf: make([]byte, 3)},
		{Addr: 2, Buf: make([]byte, 1)},
	}
	require.NoError(t, ctx.ReadBatch(reqs))
	assert.Equal(t, []byte{0x10, 0x11}, reqs[0].Buf)
	assert.Equal(t, []byte{0x15, 0x16, 0x17}, reqs[1].Buf)
	assert.Equal(t, []byte{0x12}, reqs[2].Buf)
}

func TestReadBatchEmpty(t *testing.T) {
	ctx := NewReadContext(bytes.NewReader([]byte{0xAA}))
	assert.NoError(t, ctx.ReadBatch(nil))
	assert.NoError(t, ctx.ReadBatch([]ReadReq{}))
}

type batchSpy struct {
	io.ReadSeeker
	calls int
}

func (b *batchSpy) ReadBatch(reqs []ReadReq) error {
	b.calls++
	for i := range reqs {
		// fill with 0xFF so test can distinguish from fallback path
		for j := range reqs[i].Buf {
			reqs[i].Buf[j] = 0xFF
		}
	}
	return nil
}

func TestReadBatchDispatchesToInterface(t *testing.T) {
	spy := &batchSpy{ReadSeeker: bytes.NewReader([]byte{0x00, 0x00})}
	ctx := NewReadContext(spy)
	reqs := []ReadReq{{Addr: 0, Buf: make([]byte, 1)}}
	require.NoError(t, ctx.ReadBatch(reqs))
	assert.Equal(t, 1, spy.calls)
	assert.Equal(t, byte(0xFF), reqs[0].Buf[0])
}

func TestCollector(t *testing.T) {
	data := []byte{0xA0, 0xA1, 0xA2, 0xA3, 0xA4}
	ctx := NewReadContext(bytes.NewReader(data))
	c := NewCollector(ctx)

	b1 := c.Add(0, 2)
	b2 := c.Add(3, 2)
	assert.Equal(t, 2, c.Len())
	require.NoError(t, c.Flush())
	assert.Equal(t, []byte{0xA0, 0xA1}, b1)
	assert.Equal(t, []byte{0xA3, 0xA4}, b2)
	assert.Equal(t, 0, c.Len(), "Flush should reset")
}

func TestErrors(t *testing.T) {
	var errs Errors
	assert.False(t, errs.HasFatal())
	assert.Equal(t, "<no errors>", errs.Error())

	errs.Add("A.B", 0x10, io.EOF)
	errs.Add("A.C", 0x20, io.ErrUnexpectedEOF)

	assert.True(t, errs.HasFatal())
	assert.Contains(t, errs.Error(), "A.B")
	assert.Contains(t, errs.Error(), "A.C")
}
