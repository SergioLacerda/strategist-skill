//go:build spec

package spec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyncTestStylesDocsIsIdempotent guards against the sync appending another
// "%" (or otherwise drifting) on every run.
func TestSyncTestStylesDocsIsIdempotent(t *testing.T) {
	root := repoRoot(t)
	doc := filepath.Join(t.TempDir(), "test-styles.md")
	if err := os.WriteFile(doc, []byte(readFile(t, filepath.Join(root, "docs", "test-styles.md"))), 0o644); err != nil {
		t.Fatal(err)
	}
	coverageDir := t.TempDir()
	cache := os.Getenv("GOCACHE")
	if cache == "" {
		cache = t.TempDir()
	}

	run := func() string {
		cmd := exec.Command("bash", "scripts/sync-test-styles-docs.sh", coverageDir, cache)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "TEST_STYLES_DOC="+doc, "GOCACHE="+cache)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sync failed: %v\n%s", err, out)
		}
		return readFile(t, doc)
	}

	first := run()
	second := run()
	if first != second {
		t.Fatalf("sync is not idempotent:\n--- first\n%s\n--- second\n%s", first, second)
	}
	if strings.Contains(first, "%%") {
		t.Fatalf("doc contains doubled percent sign after sync")
	}
	if strings.Contains(first, "measures 0.0%") {
		t.Fatalf("treasure-chest coverage was not measured from the current package path")
	}
}
