//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptfooManualEvidenceRunbookPreservesExternalBoundary(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "docs", "runbooks", "promptfoo-manual-evidence.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"make eval-promptfoo",
		"endpoint preflight",
		"redact",
		"report identity",
		"retention",
		"not a CI prerequisite",
		"does not certify a live conformance row",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing required manual-evidence boundary %q", path, needle)
		}
	}
}
