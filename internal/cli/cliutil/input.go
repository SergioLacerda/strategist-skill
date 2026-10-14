package cliutil

import (
	"bytes"
	"fmt"
	"io"
)

// utf8ByteOrderMark is the prefix some editors and Windows PowerShell 5.1
// (`Set-Content -Encoding utf8`, `>`) write in front of UTF-8 text.
var utf8ByteOrderMark = []byte{0xEF, 0xBB, 0xBF}

// ReadInput reads a whole JSON or YAML payload from r (usually stdin) and
// drops a leading UTF-8 byte order mark, so a payload written by such a tool
// decodes exactly as the same payload without the mark.
func ReadInput(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	return bytes.TrimPrefix(raw, utf8ByteOrderMark), nil
}
