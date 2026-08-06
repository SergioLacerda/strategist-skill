package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunArtifactCheckScenario_MissingFile(t *testing.T) {
	res := &ScenarioResult{}
	s := Scenario{Input: Input{Params: map[string]any{"path": filepath.Join(t.TempDir(), "does-not-exist.md")}}}
	runArtifactCheckScenario(s, res)
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation for a missing file, got %d: %+v", len(res.Violations), res.Violations)
	}
	if res.Violations[0].AssertionType != AssertArtifactExists {
		t.Fatalf("unexpected assertion type: %v", res.Violations[0].AssertionType)
	}
}

func TestRunArtifactCheckScenario_MustParseYAML_Invalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("not: [valid\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res := &ScenarioResult{}
	s := Scenario{Input: Input{Params: map[string]any{"path": path, "must_parse_yaml": true}}}
	runArtifactCheckScenario(s, res)
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation for invalid YAML, got %d: %+v", len(res.Violations), res.Violations)
	}
	if res.Violations[0].Message != "artifact is not valid YAML" {
		t.Fatalf("unexpected message: %q", res.Violations[0].Message)
	}
}

func TestRunArtifactCheckScenario_MustParseYAML_MissingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "good.yaml")
	if err := os.WriteFile(path, []byte("foo: bar\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res := &ScenarioResult{}
	s := Scenario{Input: Input{Params: map[string]any{
		"path": path, "must_parse_yaml": true, "must_contain_key": "missing_key",
	}}}
	runArtifactCheckScenario(s, res)
	if len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation for the missing key, got %d: %+v", len(res.Violations), res.Violations)
	}
	if res.Violations[0].Message != "artifact missing required top-level key" {
		t.Fatalf("unexpected message: %q", res.Violations[0].Message)
	}
}

func TestRunArtifactCheckScenario_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "good.yaml")
	if err := os.WriteFile(path, []byte("foo: bar\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res := &ScenarioResult{}
	s := Scenario{Input: Input{Params: map[string]any{
		"path": path, "must_parse_yaml": true, "must_contain_key": "foo",
	}}}
	runArtifactCheckScenario(s, res)
	if len(res.Violations) != 0 {
		t.Fatalf("expected no violations, got %+v", res.Violations)
	}
}
