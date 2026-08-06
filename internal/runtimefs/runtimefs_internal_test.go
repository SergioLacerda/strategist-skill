package runtimefs

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// faultyGzTempFile is a fault-injecting fake of gzTempFile: WriteGzJSON
// never gets a handle to the real *os.File it creates, and a plain
// *os.File.Close() on a regular local file has no realistic black-box
// failure trigger — this fake lets a test force that branch directly.
type faultyGzTempFile struct {
	closeErr error
}

func (f *faultyGzTempFile) Write(p []byte) (int, error) { return len(p), nil }

func (f *faultyGzTempFile) Close() error { return f.closeErr }

func TestWriteGzJSON_FileCloseErrorPropagates(t *testing.T) {
	origCreate := createGzTempFile
	t.Cleanup(func() { createGzTempFile = origCreate })
	createGzTempFile = func(_ string) (gzTempFile, error) {
		return &faultyGzTempFile{closeErr: errors.New("close boom")}, nil
	}

	path := filepath.Join(t.TempDir(), "artifact.gz")
	err := WriteGzJSON(path, map[string]string{"k": "v"})
	if err == nil || !strings.Contains(err.Error(), "file close") {
		t.Fatalf("expected a file-close error, got %v", err)
	}
}
