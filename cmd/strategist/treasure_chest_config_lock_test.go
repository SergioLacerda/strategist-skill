package main

import (
	"path/filepath"
	"testing"
)

func TestRefreshConfigLock_MissingActivePathWarnsWithoutPanicking(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "does-not-exist", "active.yaml")

	stderr := captureStderr(t, func() {
		refreshConfigLock(dir, activePath)
	})
	if stderr == "" {
		t.Fatal("expected a WARN line when the config lock cannot be refreshed")
	}
}
