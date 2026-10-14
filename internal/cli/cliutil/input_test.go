package cliutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInputStripsLeadingUTF8ByteOrderMark(t *testing.T) {
	raw, err := ReadInput(strings.NewReader("\xEF\xBB\xBF{\"a\":1}"))

	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, string(raw))
}

func TestReadInputKeepsInputWithoutByteOrderMark(t *testing.T) {
	raw, err := ReadInput(strings.NewReader("claim:\n  id: c1\n"))

	require.NoError(t, err)
	assert.Equal(t, "claim:\n  id: c1\n", string(raw))
}

func TestReadInputKeepsAByteOrderMarkThatIsNotLeading(t *testing.T) {
	raw, err := ReadInput(strings.NewReader("x\xEF\xBB\xBF"))

	require.NoError(t, err)
	assert.Equal(t, "x\xEF\xBB\xBF", string(raw))
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestReadInputPropagatesReadErrors(t *testing.T) {
	_, err := ReadInput(failingReader{})

	require.ErrorContains(t, err, "boom")
}
