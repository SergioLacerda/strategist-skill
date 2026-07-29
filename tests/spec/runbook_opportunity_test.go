//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRunbookOpportunityIsExplicitGateOnly verifies the runbook_opportunity
// routine (source and embedded mirror) is advisory-only: it must declare that
// it never writes a runbook file directly, that the runbook gate option is
// only offered when warranted, and that candidate creation requires its own
// explicit confirmation independent of any other gate response.
func TestRunbookOpportunityIsExplicitGateOnly(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "runbook-opportunity.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"phase: runbook_opportunity",
			"MUST NOT write a runbook file directly",
			"MUST NOT perform discovery beyond the already-normalized idea",
			"runbook: create_runbook_candidate    # only shown when runbook_opportunity.warranted=true",
			"requires its own explicit confirmation",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing explicit-gate-only term %q", path, needle)
			}
		}
	}
}

// TestRunbookOpportunityDoesNotClaimADROrClosureRole verifies the routine
// explicitly defers ADR-worthiness to Opportunity Attack and card
// closure/movement to Critical Hit, rather than growing into either role.
func TestRunbookOpportunityDoesNotClaimADROrClosureRole(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "runbook-opportunity.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"MUST NOT evaluate ADR-worthiness — that remains Opportunity Attack's responsibility",
			"MUST NOT evaluate card closure or movement — that remains Critical Hit's responsibility",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing role-deferral term %q", path, needle)
			}
		}
	}
}

// TestRunbookCandidateNeverWritesCanonicalDirectly verifies
// sniper_runbook_opportunity's candidate action can only produce a
// reviewable candidate — never a direct write to the canonical
// docs/runbooks/ tree — and that promotion to canonical requires human
// acceptance.
func TestRunbookCandidateNeverWritesCanonicalDirectly(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "runbook-opportunity.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"writing directly to docs/runbooks/<slug>.md from this phase",
			"promoting the candidate to canonical without human acceptance",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing candidate-forbidden term %q", path, needle)
			}
		}
	}
}

// TestOpportunityAttackInheritsRunbookSignals verifies the planned migration
// (strategist-ability-taxonomy-reorg T5, Decision D3): Opportunity Attack
// must carry an equivalent runbook_worthy activation criterion and its own
// OA-RUNBOOK side quest, while still keeping ADR, runbook, and chest
// evaluation as independent outputs that never claim card-closure (Critical
// Hit's exclusive remit).
func TestOpportunityAttackInheritsRunbookSignals(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "opportunity-attack.yaml"),
	} {
		content := readFile(t, path)
		for _, needle := range []string{
			"runbook_worthy",
			"OA-RUNBOOK-{mission_id}",
			"Opportunity Attack MUST NOT evaluate implementation completion or move cards to done/",
			"Critical Hit is the only route that may close pending/refined cards into done/",
		} {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing post-migration runbook declaration %q", path, needle)
			}
		}
	}
}

// TestCriticalHitRemainsClosureOnlyAfterRunbookOpportunityAddition is a
// regression guard: Critical Hit must not gain runbook-worthiness
// evaluation as a side effect of the runbook_opportunity routine.
func TestCriticalHitRemainsClosureOnlyAfterRunbookOpportunityAddition(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "critical-hit.yaml"),
	} {
		content := readFile(t, path)
		if strings.Contains(strings.ToLower(content), "runbook") {
			t.Fatalf("%s should not reference runbooks — closure/move only", path)
		}
	}
}

// TestPersonasExposeRunbookOpportunityGateKey verifies both personas define
// the runbook_opportunity template key used to surface the runbook candidate
// option, and that it is framed as an independent confirmation rather than a
// default action.
func TestPersonasExposeRunbookOpportunityGateKey(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "pragmatic.yaml"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults", "personas", "epic.yaml"),
	} {
		content := readFile(t, path)
		if !strings.Contains(content, "runbook_opportunity: >") {
			t.Fatalf("%s missing runbook_opportunity template key", path)
		}
		if !strings.Contains(content, "candidate_id") || !strings.Contains(content, "candidate_title") {
			t.Fatalf("%s runbook_opportunity template missing candidate_id/candidate_title placeholders", path)
		}
	}
}

// TestDocsRunbooksPolicyDeclaresSourceFirstProvenance verifies the runbook
// ownership policy documents docs/runbooks/ as canonical, requires provenance
// metadata on any runtime-optimized artifact, and defines a deterministic
// source-vs-runtime mismatch outcome.
func TestDocsRunbooksPolicyDeclaresSourceFirstProvenance(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "docs", "runbooks", "README.md")
	content := readFile(t, path)
	for _, needle := range []string{
		"canonical",
		"derived cache",
		"source_hash",
		"freshness: fresh|stale|unknown",
		"prefer canonical source",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing runbook policy term %q", path, needle)
		}
	}
}
