package integrity

import (
	"path/filepath"
	"testing"
)

// TestHashFile_DoesNotExist is a white-box test for hashFile's own
// "does not exist" branch — reachable directly (path was never Stat-gated
// by a caller), unlike via WriteLock/Check, which always Stat the config
// path before ever calling hashFile.
func TestHashFile_DoesNotExist(t *testing.T) {
	_, err := hashFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a nonexistent path, got nil")
	}
}
