package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	jsonlScannerInitialSize = 64 * 1024
	jsonlScannerMaxSize     = 1024 * 1024
)

func newJSONLScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, jsonlScannerInitialSize), jsonlScannerMaxSize)
	return scanner
}

func jsonlScannerErr(scanner *bufio.Scanner, context string) error {
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func closeFileWithContext(f *os.File, err *error, context string) {
	if closeErr := f.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("%s: %w", context, closeErr)
	}
}

// CountUnreadableJSONLines counts the non-empty lines of a JSONL file that are
// not a JSON object. The ledger readers skip such lines silently; an integrity
// check needs the number. A missing file has none.
func CountUnreadableJSONLines(path string) (n int, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: ledger path is owned by the runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("open jsonl file: %w", err)
	}
	defer closeFileWithContext(f, &err, "close jsonl file")
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var obj map[string]json.RawMessage
		if len(line) > 0 && json.Unmarshal(line, &obj) != nil {
			n++
		}
	}
	return n, jsonlScannerErr(scanner, "scan jsonl file")
}
