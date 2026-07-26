package telemetry

import (
	"bufio"
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
