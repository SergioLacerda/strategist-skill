//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// legacyMissionStatusTokens are the two retired mission_status values (D1). Both are
// matched with a word boundary so gate_approval_missing — a distinct, still-valid
// reason code for "the gate has not reached gate_analysis_accepted yet" — is not a
// false positive.
var legacyMissionStatusTokens = []*regexp.Regexp{
	regexp.MustCompile(`\bgate_approval\b`),
	regexp.MustCompile(`\bexecution_done\b`),
}

// missionStatusVocabularyAllowlist lists files allowed to mention the legacy tokens:
// contracts/machine/mission-status.yaml is the canonical table itself, and it
// documents the retired->canonical mapping in its legacy_compatibility section by
// design (D1's own fix artifact, not a leftover site).
func missionStatusVocabularyAllowlisted(path string) bool {
	return filepath.Base(path) == "mission-status.yaml"
}

// TestNoLegacyMissionStatusTokensInShippedTrees is the D1 regression guard: no file
// under strategist/ or internal/embed/defaults/ may use gate_approval or
// execution_done as a mission_status token. Historical mentions in .analysis/ or
// archived ADRs are out of scope (workspace-local, not shipped).
func TestNoLegacyMissionStatusTokensInShippedTrees(t *testing.T) {
	t.Parallel()

	roots := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults"),
	}

	for _, root := range roots {
		root := root
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			ext := filepath.Ext(path)
			if ext != ".md" && ext != ".yaml" {
				return nil
			}
			if missionStatusVocabularyAllowlisted(path) {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			content := string(data)
			for _, pattern := range legacyMissionStatusTokens {
				if pattern.MatchString(content) {
					t.Errorf("%s contains legacy mission_status token %q — use the canonical vocabulary in contracts/machine/mission-status.yaml", path, pattern.String())
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
