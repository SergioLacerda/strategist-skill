package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloseFileWithContext_ReturnsErrorOnDoubleClose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "double-close"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	var outerErr error
	closeFileWithContext(f, &outerErr, "close outcomes file")
	if outerErr == nil {
		t.Fatal("expected error on double close, got nil")
	}
}

func TestCloseFileWithContext_DoesNotOverwriteExistingError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "existing-err"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	sentinel := errors.New("prior error")
	outerErr := sentinel
	closeFileWithContext(f, &outerErr, "close outcomes file")
	if !errors.Is(outerErr, sentinel) {
		t.Fatalf("expected prior error preserved, got: %v", outerErr)
	}
}

func TestJSONLScannerErr_ReportsTooLongLine(t *testing.T) {
	t.Parallel()
	// A line exceeding the scanner's 1MB max triggers bufio.ErrTooLong.
	huge := strings.Repeat("x", 2*1024*1024)
	scanner := newJSONLScanner(strings.NewReader(huge))
	for scanner.Scan() {
		// drain
	}
	err := jsonlScannerErr(scanner, "scan test")
	if err == nil {
		t.Fatal("expected scanner error for oversized line, got nil")
	}
	if !strings.Contains(err.Error(), "scan test") {
		t.Fatalf("expected error to include context, got: %v", err)
	}
}
