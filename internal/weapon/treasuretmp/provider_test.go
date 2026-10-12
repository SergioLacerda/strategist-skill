package treasuretmp

import (
	"os"
	"path/filepath"
	"testing"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
)

func writeIndexFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte("schema_version: '1'\njewels:\n  - id: jewel-one\n    chest_id: source\n    kind: decision\n    statement: repair telemetry gate\n    source_refs: [docs/source.md]\n    trust: T3\n    status: accepted\n    reviewed_by: human\n    score:\n      value: 10\n    applicability:\n      scope: [execution]\n      applies_when: [telemetry]\n      avoid_when: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "potions.yaml"), []byte("schema_version: '1'\npotions:\n  - id: runbook-one\n    chest_id: runbooks\n    runbook_ref: docs/runbooks/one.md\n    when_to_use: repair telemetry gate\n    trust: T2\n    status: accepted\n    source_refs: [docs/runbooks/one.md]\n    reviewed_by: human\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSearchMapsRootPotionsAndHonorsLimit(t *testing.T) {
	response, err := Search(Request{Root: writeIndexFixture(t), Concepts: []string{"telemetry"}, TokenBudget: 1})
	if err != nil {
		t.Fatal(err)
	}
	if response.Envelope.Provider != ProviderName || response.Envelope.FallbackState != "none" {
		t.Fatalf("bad envelope: %#v", response.Envelope)
	}
	if len(response.Items) != 1 || response.Items[0].Kind != "jewel" {
		t.Fatalf("unexpected items: %#v", response.Items)
	}
}

func TestPrepareAndStatusReportHealthyWorkspace(t *testing.T) {
	root := writeIndexFixture(t)
	prepared, err := Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.SourcesRead != 0 || prepared.ItemsIndexed != 2 {
		t.Fatalf("unexpected prepare: %#v", prepared)
	}
	if status := Status(root); !status.Healthy {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestSearchMatchesJewelsAndPotionsWithoutFilters(t *testing.T) {
	response, err := Search(Request{Root: writeIndexFixture(t), TokenBudget: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 2 || response.Items[0].Kind != "jewel" {
		t.Fatalf("unexpected response: %#v", response.Items)
	}
}

func TestLoadErrorsPropagateFromMalformedManifests(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte("schema_version: [bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(root); err == nil {
		t.Fatal("expected prepare error")
	}
	if _, err := Search(Request{Root: root}); err == nil {
		t.Fatal("expected search error")
	}
}

func TestScoringHelpersCoverTrustAndSources(t *testing.T) {
	for _, value := range []string{"T0", "T1", "T2", "T3", "unknown"} {
		_ = trustScore(value)
	}
	if first(nil) != "" || first([]string{"source"}) != "source" {
		t.Fatal("first source mismatch")
	}
	jewel := treasure.Jewel{ID: "id", Statement: "needle", Score: treasure.JewelScore{Value: 4}}
	if score, matched := matchJewel(jewel, []string{"needle"}); !matched || score <= 0 {
		t.Fatalf("jewel match failed: %v %v", score, matched)
	}
	if _, matched := matchPotion(treasure.Potion{WhenToUse: "other"}, []string{"needle"}); matched {
		t.Fatal("unexpected potion match")
	}
}

func TestStatusIsUnhealthyForMalformedWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "potions.yaml"), []byte("schema_version: [bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	status := Status(root)
	if status.Healthy || status.Provider != ProviderName || len(status.Limitations) == 0 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestSearchRejectsMissingRoot(t *testing.T) {
	if _, err := Search(Request{}); err == nil {
		t.Fatal("expected missing root error")
	}
}
