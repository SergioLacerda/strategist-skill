//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSelectRunbookIsWiredIntoRetrievalCascadeStage6 pins the decision that the
// pipeline invokes the select_runbook ability instead of leaving it as prose:
// Ranger runs `strategist runbook select` at Retrieval Cascade stage 6 and the
// handoff distinguishes "ran, no match" (empty list) from "never ran" (null).
func TestSelectRunbookIsWiredIntoRetrievalCascadeStage6(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		"roles/ranger.yaml":                               {"strategist runbook select", "--signal", "must_invoke"},
		"contracts/narrative/03-discovery.md":             {"Ranger MUST run `strategist runbook select", "an empty list means the command ran and nothing matched"},
		"internal_skills/ranger/SKILL.md":                 {"strategist runbook select"},
		"schemas/handoff-ranger-to-archivist.schema.yaml": {"an empty list means the command ran and nothing matched", "null means it was not run"},
	}
	for _, root := range []string{filepath.Join(repoRoot(t), "internal", "embed", "defaults"), isolatedStrategistDir(t)} {
		for rel, wants := range required {
			data, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Errorf("%s: %v", filepath.Join(root, rel), err)
				continue
			}
			flat := strings.Join(strings.Fields(string(data)), " ") // YAML and Markdown wrap lines
			for _, want := range wants {
				if !strings.Contains(flat, want) {
					t.Errorf("%s: missing %q", filepath.Join(root, rel), want)
				}
			}
		}
	}
}

// The command Ranger is told to run must exist in the CLI.
func TestSelectRunbookCommandIsInTheCommandTree(t *testing.T) {
	t.Parallel()

	golden, err := os.ReadFile(filepath.Join(repoRoot(t), "cmd", "strategist", "testdata", "command_tree.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(golden), "== strategist runbook select") {
		t.Fatal("`strategist runbook select` is not in the command tree, but the Ranger contract invokes it")
	}
}
