package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
)

func TestScenarioHasCriteria_Present(t *testing.T) {
	t.Parallel()
	dojoDir := t.TempDir()
	scenarioDir := filepath.Join(dojoDir, "my-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte("scenario: my-scenario\n"), 0o644); err != nil {
		t.Fatalf("write criteria.yaml: %v", err)
	}

	if !dojo.ScenarioHasCriteria(dojoDir, "my-scenario") {
		t.Fatal("expected ScenarioHasCriteria to report true when criteria.yaml exists")
	}
}

func TestScenarioHasCriteria_Absent(t *testing.T) {
	t.Parallel()
	dojoDir := t.TempDir()

	if dojo.ScenarioHasCriteria(dojoDir, "missing-scenario") {
		t.Fatal("expected ScenarioHasCriteria to report false when the scenario directory is absent")
	}
}

func TestScenarioHasCriteria_DirWithoutCriteriaFile(t *testing.T) {
	t.Parallel()
	dojoDir := t.TempDir()
	scenarioDir := filepath.Join(dojoDir, "empty-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}

	if dojo.ScenarioHasCriteria(dojoDir, "empty-scenario") {
		t.Fatal("expected ScenarioHasCriteria to report false when criteria.yaml is missing")
	}
}
