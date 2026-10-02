package refinement

import (
	"os"
	"path/filepath"
	"testing"
)

func writePackage(t *testing.T, files map[string]string) string {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "refined", "m1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

func TestReadSideQuestsFromFencedBlockAndFrontmatter(t *testing.T) {
	base := writePackage(t, map[string]string{
		"tasks.md":    "# t\n```yaml\nside_quests_approved:\n  - {id: OA-ADR-m1, description: d, strategy: execute_together, status: sq_pending}\n```\n",
		"analysis.md": "---\nside_quests_approved:\n  - id: SQ-001\n    status: sq_backlog\n---\nbody\n",
	})
	got, err := ReadSideQuests(base, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 side quests, got %+v", got)
	}
	if q, ok := FindSideQuest(got, "OA-ADR-m1"); !ok || q.Strategy != "execute_together" {
		t.Fatalf("OA-ADR not found: %+v", got)
	}
	if _, ok := FindSideQuest(got, "OA-ADR-other"); ok {
		t.Fatal("foreign id must not be found")
	}
}

func TestReadSideQuestsToleratesBrokenYAMLAndMissingPackage(t *testing.T) {
	base := writePackage(t, map[string]string{"tasks.md": "```yaml\nside_quests_approved: [unclosed\n```\n"})
	got, err := ReadSideQuests(base, "m1")
	if err != nil || len(got) != 0 {
		t.Fatalf("broken yaml must be skipped: %v %+v", err, got)
	}
	got, err = ReadSideQuests(t.TempDir(), "absent")
	if err != nil || len(got) != 0 {
		t.Fatalf("missing package must declare none: %v %+v", err, got)
	}
}
