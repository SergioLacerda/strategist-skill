//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCriticalHitSupportsEvidenceGatedClosureIntoDone verifies Critical Hit's
// narrative and machine contracts describe a closure move into done/ that
// requires an explicit completion claim and a supplied evidence summary, and
// never infers completion from code alone. Close Card was folded into
// Critical Hit as a second mode rather than kept as a separate route.
func TestCriticalHitSupportsEvidenceGatedClosureIntoDone(t *testing.T) {
	t.Parallel()

	narrativePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	}
	for _, path := range narrativePaths {
		content := readFile(t, path)
		for _, needle := range []string{
			"Closure move",
			"`<base_path>/done/<id>`",
			"evidence summary is available",
			"never infers completion",
			"Stale Card Detection",
			"Discovery (Ranger)",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing closure-move term %q", path, needle)
			}
		}
		if strings.Contains(content, "close-card.md") || strings.Contains(content, "12-close-card") {
			t.Fatalf("%s still references a separate close-card file after consolidation", path)
		}
	}

	machinePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	}
	for _, path := range machinePaths {
		content := readFile(t, path)
		for _, needle := range []string{
			"closure_move",
			"evidence_summary_present: true",
			"completion_inferred_from_code_only",
			"stale_card_detection",
			"on: discovery",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing closure-move term %q", path, needle)
			}
		}
	}
}

// TestCloseCardIsNotASeparateRoute verifies close-card.yaml and
// 12-close-card.md were removed as standalone files (source and embedded
// defaults) after Close Card was consolidated into Critical Hit, and that no
// other contract or skill.yaml still refers to close_card as an independent
// resolution path.

// TestCloseCardIsNotASeparateRoute verifies close-card.yaml and
// 12-close-card.md were removed as standalone files (source and embedded
// defaults) after Close Card was consolidated into Critical Hit, and that no
// other contract or skill.yaml still refers to close_card as an independent
// resolution path.
func TestCloseCardIsNotASeparateRoute(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	for _, rel := range []string{
		filepath.Join("internal", "embed", "defaults", "contracts", "narrative", "12-close-card.md"),
		filepath.Join("internal", "embed", "defaults", "contracts", "machine", "close-card.yaml"),
		filepath.Join("internal", "embed", "defaults", "contracts", "narrative", "12-close-card.md"),
		filepath.Join("internal", "embed", "defaults", "contracts", "machine", "close-card.yaml"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Fatalf("%s should not exist — close-card was folded into critical-hit.yaml/11-critical-hit.md", rel)
		}
	}

	for _, path := range []string{
		filepath.Join(root, "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "close_card:") {
			t.Fatalf("%s still declares close_card as a separate resolution path", path)
		}
	}
}

// TestDocumentationAppliedDoesNotTriggerClosure verifies the corrected
// lifecycle model: reaching documentation_applied at the end of a
// main_mission is documentation completion only, does not trigger a Critical
// Hit closure candidacy check, and does not imply the package should move to
// done/. A completed main_mission ending with its package in refined/ is the
// normal, expected terminal state — not a gap the pipeline auto-corrects.
// This supersedes the earlier (incorrect) auto-closure-check design.

// TestDocumentationAppliedDoesNotTriggerClosure verifies the corrected
// lifecycle model: reaching documentation_applied at the end of a
// main_mission is documentation completion only, does not trigger a Critical
// Hit closure candidacy check, and does not imply the package should move to
// done/. A completed main_mission ending with its package in refined/ is the
// normal, expected terminal state — not a gap the pipeline auto-corrects.
// This supersedes the earlier (incorrect) auto-closure-check design.
func TestDocumentationAppliedDoesNotTriggerClosure(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"is documentation completion, not implementation or validation evidence",
			"does not by itself trigger Critical Hit closure",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing corrected documentation_applied term %q", path, needle)
			}
		}
		if strings.Contains(content, "triggers Critical Hit's stale-card detection") {
			t.Fatalf("%s still contains the superseded auto-closure-check wiring", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "00-routing.md"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "closure_check") {
			t.Fatalf("%s still contains the superseded closure_check step in the Main Mission Sequence", path)
		}
		if !strings.Contains(content, "Main mission completion does not imply implementation completion") {
			t.Fatalf("%s missing the corrected main-mission-completion statement", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "main_mission_execution_complete") {
			t.Fatalf("%s still contains the superseded main_mission_execution_complete trigger", path)
		}
		if !strings.Contains(content, "insufficient_evidence") {
			t.Fatalf("%s missing insufficient_evidence list", path)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Insufficient Evidence",
			"does NOT trigger a closure candidacy check",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing corrected Critical Hit term %q", path, needle)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "skill.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(content, "Fires\n        automatically at end of main_mission execution") ||
			strings.Contains(content, "Fires automatically at end of main_mission execution") {
			t.Fatalf("%s still claims critical_hit fires automatically at end of main_mission execution", path)
		}
	}
}

// TestApprovalGateAcceptanceDoesNotAuthorizeCodeMutation verifies the approval
// gate contract explicitly states that gate acceptance approves the refined
// analysis and documentation_target items only, never code/hook/config/test
// mutation — closing the drift where a prior mission treated gate acceptance
// as permission to edit source files directly.

// TestCriticalHitClosureCrossReferencesEvidenceState verifies the critical-hit
// narrative and machine contracts tie closure evidence to Scout's evidence_state
// vocabulary without weakening the existing Insufficient Evidence invariants.
func TestCriticalHitClosureCrossReferencesEvidenceState(t *testing.T) {
	t.Parallel()

	narrativePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "11-critical-hit.md"),
	}
	for _, path := range narrativePaths {
		content := readFile(t, path)
		if !strings.Contains(content, "evidence_state: explicit") {
			t.Fatalf("%s missing evidence_state cross-reference", path)
		}
	}

	machinePaths := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	}
	for _, path := range machinePaths {
		content := readFile(t, path)
		if !strings.Contains(content, "evidence_state: explicit") {
			t.Fatalf("%s missing evidence_state cross-reference", path)
		}
	}
}

// TestBrainstormingProviderDeclaresOnlyCreativeSubtypeSupport verifies the
// brainstorming provider manifest declares creative-subtype support only.
// It must NOT claim evaluation/diagnostic/closure_evidence adapter support:
// those subtypes bypass this weapon entirely and resolve to
// internal_skills/ranger (see 00-routing.md § Discovery Weapon Resolution by
// Subtype). A prior version of this test required the adapter claims —
// that assumption was falsified by a live invocation showing brainstorming's
// own SKILL.md has no adaptive behavior for those subtypes at all.
// .strategist/ is a generated runtime artifact, so this only asserts the
// embedded-defaults copy — the canonical source for what strategist install stamps
// into a workspace.
