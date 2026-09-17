package authorization

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAllowedAnalysisTargetKeepsLiveEvidenceUnverified(t *testing.T) {
	root := writeAuthorizationFixture(t)
	report, err := Build(Request{Root: root, Target: ".analysis/refined/mission/tasks.md", ApprovalGate: "accepted"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if report.Decision != "allowed" || report.ExitClass != "success" {
		t.Fatalf("unexpected decision: %+v", report)
	}
	if report.ReasonCode != "authorization_dimensions_satisfied" {
		t.Fatalf("unexpected reason: %s", report.ReasonCode)
	}
	if dimension := findDimension(report, "live_provider"); dimension.Status != "unknown" || dimension.Required {
		t.Fatalf("live provider evidence was overstated: %+v", dimension)
	}
}

func TestBuildDeniesForbiddenPlanningPath(t *testing.T) {
	root := writeAuthorizationFixture(t)
	report, err := Build(Request{Root: root, Target: "docs/plans/next.md", ApprovalGate: "accepted"})
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("expected ErrDenied, got %v", err)
	}
	if report.ReasonCode != "forbidden_planning_path" || report.ExitClass != "denied" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestBuildBlocksWithoutApproval(t *testing.T) {
	root := writeAuthorizationFixture(t)
	report, err := Build(Request{Root: root, Target: ".analysis/refined/mission/tasks.md"})
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
	if report.ReasonCode != "approval_gate_not_accepted" || report.Decision != "blocked" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestBuildBlocksBindingMismatch(t *testing.T) {
	root := writeAuthorizationFixture(t)
	lock := filepath.Join(root, "plugins.lock")
	raw, err := os.ReadFile(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte(string(raw)+"\n  # fixture remains structurally valid but duplicate discovery binding\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The fixture mutation above is intentionally harmless; replace the active
	// provider so the persisted binding parity is the failing dimension.
	active := filepath.Join(root, "active.yaml")
	activeRaw, err := os.ReadFile(active)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte(string(activeRaw)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// BuildRoleInvocationPlan itself is the authoritative binding check; an
	// absent lock node produces a stable blocked result.
	if err := os.WriteFile(lock, []byte("schema_version: strategist-plugin-lock-file/v1\nlock:\n  nodes: []\nbindings: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Build(Request{Root: root, Target: ".analysis/refined/mission/tasks.md", ApprovalGate: "accepted"})
	if !errors.Is(err, ErrBlocked) || report.ReasonCode != "role_provider_binding_invalid" {
		t.Fatalf("unexpected binding result: report=%+v err=%v", report, err)
	}
}

func TestBuildBlocksStaleCompiledRuntime(t *testing.T) {
	root := writeAuthorizationFixture(t)
	if err := os.MkdirAll(filepath.Join(root, ".compiled"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".compiled", ".config.gz"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Build(Request{Root: root, Target: ".analysis/refined/mission/tasks.md", ApprovalGate: "accepted"})
	if !errors.Is(err, ErrStale) || report.ExitClass != "stale" || report.ReasonCode != "compiled_runtime_stale" {
		t.Fatalf("unexpected stale result: report=%+v err=%v", report, err)
	}
}

func findDimension(report Report, name string) Dimension {
	for _, dimension := range report.Dimensions {
		if dimension.Name == name {
			return dimension
		}
	}
	return Dimension{}
}

func writeAuthorizationFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "roles"), 0o700); err != nil {
		t.Fatal(err)
	}
	write("active.yaml", "mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\n")
	write("roles/default.yaml", "discovery: ranger\nrefinement: archivist\nexecution: sniper\n")
	write("plugins.lock", `schema_version: strategist-plugin-lock-file/v1
lock:
  nodes:
    - id: brainstorming
      kind: adapter_contract
      digest: sha256:brainstorming
    - id: openspec-propose
      kind: adapter_contract
      digest: sha256:openspec
    - id: sniper
      kind: adapter_contract
      digest: sha256:sniper
bindings:
  - slot: discovery
    installed_instance_id: brainstorming
    status: active
  - slot: refinement
    installed_instance_id: openspec-propose
    status: active
  - slot: execution
    installed_instance_id: sniper
    status: active
`)
	return root
}
