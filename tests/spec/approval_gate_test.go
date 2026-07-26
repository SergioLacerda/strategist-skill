//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovalGateContractDefinesEmitOnShow(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "emit_on_show") {
			t.Fatalf("%s missing \"emit_on_show\" — approval gate must log when shown to user", p)
		}
	}
}

func TestContextEnrichmentDefinesTreasureChestLoadedNone(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "context-enrichment.yaml"),
	} {
		content := readFile(t, p)
		if !strings.Contains(content, "treasure_chest_loaded none") {
			t.Fatalf("%s missing \"treasure_chest_loaded none\" emit for empty chest list", p)
		}
	}
}

func TestComplianceSummaryDefinesPhaseCounters(t *testing.T) {
	t.Parallel()
	for _, p := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "compliance-summary.yaml"),
	} {
		content := readFile(t, p)
		for _, needle := range []string{"expected_phases", "executed_phases"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q counter field in compliance emit_format", p, needle)
			}
		}
	}
}

// --- Provider discovery conformance tests ---

// providerBootstrapFiles lists all provider bootstrap surfaces that must declare
// Strategist runtime discovery semantics.

func TestApprovalGateContractUsesReviewGateSemantics(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	paths := []string{
		filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml"),
	}

	for _, path := range paths {
		content := readFile(t, path)
		assertNoToken(t, path, "plan_only")
		for _, needle := range []string{"analysis_accepted", "revision_requested", "rejected"} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing review gate response %q", path, needle)
			}
		}
	}
}

// TestApprovalGateAcceptanceDoesNotAuthorizeCodeMutation verifies the approval
// gate contract explicitly states that gate acceptance approves the refined
// analysis and documentation_target items only, never code/hook/config/test
// mutation — closing the drift where a prior mission treated gate acceptance
// as permission to edit source files directly.
func TestApprovalGateAcceptanceDoesNotAuthorizeCodeMutation(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"Gate Acceptance Is Not Code Mutation Approval",
			"implementation_handoff",
			"not as a separate mission status",
			"requires a separate coding task outside",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing gate/code-mutation term %q", path, needle)
			}
		}
		assertNoToken(t, path, "implementation_handoff_ready")
	}

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "agent-protocol.md"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "Never treat Strategist Approval Gate acceptance") {
			t.Fatalf("%s missing NEVER DO bullet for gate-acceptance/code-mutation confusion", path)
		}
	}
}

// TestDriftPatternsIncludeApprovalGateCodeExecutionConfusion verifies the
// drift-patterns template declares the approval_gate_code_execution_confusion
// pattern so the agent self-corrects instead of re-reading full governance.
func TestDriftPatternsIncludeApprovalGateCodeExecutionConfusion(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "templates", "domain", "identity", "drift-patterns.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"id: approval_gate_code_execution_confusion",
			"never authorizes code mutation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing drift pattern term %q", path, needle)
			}
		}
	}
}

// TestEvidencePackContractDefinesNonBlockingEmptyState verifies the Evidence Pack
// contract (Track T-A) declares its fields and the empty-state non-blocking behavior,
// and never turns evidence packs into a raw-chest-load or new retrieval unit.
