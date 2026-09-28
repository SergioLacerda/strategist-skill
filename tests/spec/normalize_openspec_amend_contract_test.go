//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRefinementContractNamesTheAmendMode pins that a change to a package after the
// gate goes only through `normalize-openspec --amend` with a recorded authorization
// reference, never by hand (DEC-006, DEC-007, DEC-008).
func TestRefinementContractNamesTheAmendMode(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	narrative := filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "04-refinement.md")
	content := readFile(t, narrative)
	for _, needle := range []string{
		"--amend --amends <previous_change_id> --authorization-ref",
		"never by hand",
		".amendments/",
		"analysis.md",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing amendment clause term %q", narrative, needle)
		}
	}
	registry := filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "mechanisms.yaml")
	mechanisms := readFile(t, registry)
	i := strings.Index(mechanisms, "id: normalize_openspec")
	if i < 0 || !strings.Contains(mechanisms[i:i+900], "--amend") {
		t.Fatalf("%s: normalize_openspec must name the --amend mode in its summary or when_to_use", registry)
	}
}
