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
