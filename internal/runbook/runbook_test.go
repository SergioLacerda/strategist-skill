package runbook

import (
	"strings"
	"testing"
)

func validSidecarYAML() string {
	return `
schema_version: "1"
runbook_id: example-runbook
runbook_type: analytical
source_doc: docs/runbooks/example-runbook.md
applies_when:
  - "symptom: build fails after dependency upgrade"
objective: Diagnose a post-upgrade build failure.
preconditions:
  - "go.mod has a recent dependency bump"
analysis:
  - "Bisect the failing dependency via go.sum diff"
decision_gates:
  - id: root-cause-confirmed
    statement: The failing dependency is identified with reproducing evidence.
checks:
  - id: reproduced-locally
    level: mandatory
  - id: changelog-reviewed
    level: recommended
metadata:
  version: 1
  reviewed_at: "2026-08-03"
`
}

func TestParseSidecar_Valid(t *testing.T) {
	t.Parallel()
	rb, err := ParseSidecar([]byte(validSidecarYAML()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rb.RunbookID != "example-runbook" {
		t.Fatalf("expected runbook_id example-runbook, got %q", rb.RunbookID)
	}
	if rb.RunbookType != RunbookTypeAnalytical {
		t.Fatalf("expected runbook_type analytical, got %q", rb.RunbookType)
	}
	if len(rb.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(rb.Checks))
	}
}

func TestParseSidecar_InvalidYAML(t *testing.T) {
	t.Parallel()
	_, err := ParseSidecar([]byte("not: [valid"))
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestParseSidecar_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	_, err := ParseSidecar([]byte(`
schema_version: "1"
runbook_id: incomplete
runbook_type: analytical
`))
	if err == nil {
		t.Fatal("expected error for missing source_doc/objective/applies_when")
	}
	for _, want := range []string{"source_doc", "objective", "applies_when"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %q, got: %v", want, err)
		}
	}
}

func TestValidateRunbook_InvalidRunbookType(t *testing.T) {
	t.Parallel()
	rb, err := ParseSidecar([]byte(strings.Replace(validSidecarYAML(), "runbook_type: analytical", "runbook_type: bogus", 1)))
	if err == nil {
		t.Fatal("expected error for invalid runbook_type")
	}
	_ = rb
}

func TestValidateRunbook_CascadesToNestedCheckAndGate(t *testing.T) {
	t.Parallel()
	bad := strings.Replace(validSidecarYAML(), "level: mandatory", "level: bogus", 1)
	_, err := ParseSidecar([]byte(bad))
	if err == nil {
		t.Fatal("expected error to cascade from an invalid nested check level")
	}
	if !strings.Contains(err.Error(), "check_invalid") {
		t.Errorf("expected check_invalid in error, got: %v", err)
	}
}
