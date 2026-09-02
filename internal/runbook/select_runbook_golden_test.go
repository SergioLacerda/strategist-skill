package runbook

import (
	"os"
	"path/filepath"
	"testing"
)

// loadSidecarFixture parses a real docs/runbooks/<name>.runbook.yaml sidecar
// from disk. Tests in this file run with the package directory as their
// working directory (go test's default), so the fixture path climbs two
// levels back to the repo root.
func loadSidecarFixture(t *testing.T, name string) Runbook {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "runbooks", name)
	data, err := os.ReadFile(path) //nolint:gosec // G304: fixed test fixture path, not user input
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	rb, err := ParseSidecar(data)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return rb
}

// TestSelect_GoldenAgainstRealRunbookFixtures is a golden/example-based
// corpus tying Select's matching behavior to the real
// docs/runbooks/*.runbook.yaml definitions rather than hand-written
// fixtures (see select_runbook_test.go for the mechanics-only coverage).
// Each case names the real sidecar file it loads, a mission signal set,
// and whether that runbook should be selected. Several positive cases are
// deliberately worded so that the exact signal string is NOT a literal
// substring of the runbook's applies_when text — those cases only pass
// because of the controlled signal vocabulary (signal_vocabulary.go)
// normalizing both sides to a shared CanonicalSignal; they would fail
// under the old pure-substring matcher and exist to prove the vocabulary
// layer is doing real work, not just decoration.
func TestSelect_GoldenAgainstRealRunbookFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		sidecarFile string
		signals     MissionSignals
		wantMatch   bool
	}{
		{
			name:        "ci_test_failure: operator synonym matches via vocabulary, not substring",
			sidecarFile: "verifying-test-failures.runbook.yaml",
			signals:     MissionSignals{"flaky test observed in the nightly run"},
			wantMatch:   true,
		},
		{
			name:        "ci_test_failure: unrelated dependency signal does not match",
			sidecarFile: "verifying-test-failures.runbook.yaml",
			signals:     MissionSignals{"dependency bump landed on main"},
			wantMatch:   false,
		},
		{
			name:        "dependency_upgrade: reworded real-phrase signal matches via vocabulary",
			sidecarFile: "verifying-dependency-upgrades.runbook.yaml",
			signals:     MissionSignals{"npm audit fix --force reported a breaking change on a direct dependency"},
			wantMatch:   true,
		},
		{
			name:        "dependency_upgrade: release-tooling signal does not match",
			sidecarFile: "verifying-dependency-upgrades.runbook.yaml",
			signals:     MissionSignals{"release tooling looks off after a tag push"},
			wantMatch:   false,
		},
		{
			name:        "release_tool_version_drift: reworded deprecation-warning signal matches via vocabulary",
			sidecarFile: "release-tool-version-drift.runbook.yaml",
			signals:     MissionSignals{"a deprecation warning showed up in the CI logs for an untouched config"},
			wantMatch:   true,
		},
		{
			name:        "release_tool_version_drift: concurrent-session signal does not match",
			sidecarFile: "release-tool-version-drift.runbook.yaml",
			signals:     MissionSignals{"two sessions collided writing the same file"},
			wantMatch:   false,
		},
		{
			name:        "concurrent_session_collision: reworded signal matches via vocabulary",
			sidecarFile: "concurrent-session-sniper-collision.runbook.yaml",
			signals:     MissionSignals{"we saw a sniper materializing to the same file during a concurrent run"},
			wantMatch:   true,
		},
		{
			name:        "concurrent_session_collision: breaking-change signal does not match",
			sidecarFile: "concurrent-session-sniper-collision.runbook.yaml",
			signals:     MissionSignals{"breaking change in a direct dependency after upgrade"},
			wantMatch:   false,
		},
		{
			name:        "provider_invocation_failure: exact error code matches",
			sidecarFile: "role-invocation-failed.runbook.yaml",
			signals:     MissionSignals{"error=role_invocation_failed observed for slot=execution"},
			wantMatch:   true,
		},
		{
			name:        "provider_invocation_failure: unrelated complexity signal does not match",
			sidecarFile: "role-invocation-failed.runbook.yaml",
			signals:     MissionSignals{"golangci-lint flagged a gocritic violation"},
			wantMatch:   false,
		},
		{
			name:        "treasure_chest_partial_write: reworded signal matches via vocabulary",
			sidecarFile: "treasure-chest-partial-write.runbook.yaml",
			signals:     MissionSignals{"the chest command exited early and now active.yaml looks left in an inconsistent state"},
			wantMatch:   true,
		},
		{
			name:        "treasure_chest_partial_write: verifying-implemented-demands signal does not match",
			sidecarFile: "treasure-chest-partial-write.runbook.yaml",
			signals:     MissionSignals{"the user asked if this refined package is already finished"},
			wantMatch:   false,
		},
		{
			name:        "verifying_implemented_demands: reworded signal matches via vocabulary",
			sidecarFile: "verifying-implemented-demands.runbook.yaml",
			signals:     MissionSignals{"the user asked if this refined package is already finished"},
			wantMatch:   true,
		},
		{
			name:        "complexity_refactor: reworded signal matches via vocabulary",
			sidecarFile: "refactoring-for-agent-operations.runbook.yaml",
			signals:     MissionSignals{"golangci-lint flagged a gocritic violation in this package"},
			wantMatch:   true,
		},
		{
			name:        "complexity_refactor: ci test failure signal does not match",
			sidecarFile: "refactoring-for-agent-operations.runbook.yaml",
			signals:     MissionSignals{"flaky test observed in the nightly run"},
			wantMatch:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rb := loadSidecarFixture(t, tc.sidecarFile)

			selections, _, err := Select([]Runbook{rb}, tc.signals, DefaultSelectionPolicy())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotMatch := len(selections) > 0
			if gotMatch != tc.wantMatch {
				t.Fatalf("runbook %q signals %v: got matched=%v, want matched=%v (selections=%v)",
					rb.RunbookID, tc.signals, gotMatch, tc.wantMatch, selections)
			}
			if gotMatch && selections[0].Reason == "" {
				t.Fatalf("runbook %q: matched selection has empty reason", rb.RunbookID)
			}
		})
	}
}
