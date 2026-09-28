//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOpportunityAttackRunbookIsExplicitGateOnly verifies the runbook side of
// Opportunity Attack (the superseded runbook-opportunity contract's signals now
// live here) is advisory-only: it only surfaces a proposed candidate as a side
// quest, never writes canonical docs/runbooks/ content from its own action, and
// never promotes a candidate without a gate approval.
func TestOpportunityAttackRunbookIsExplicitGateOnly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "opportunity-attack.yaml")
	content := readFile(t, path)
	for _, needle := range []string{
		"status: proposed",
		"never a trusted docs/runbooks/ entry without human review",
		"writing directly to docs/runbooks/<slug>.md from *this* action",
		"promoting the candidate to canonical without a gate approval",
		"Opportunity Attack never writes docs/runbooks/ or treasure-chests.yaml directly, only surfaces the side quest",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s missing explicit-gate-only term %q", path, needle)
		}
	}
}

// TestRunbookOpportunityContractIsRetired guards the removal of the superseded
// runbook-opportunity contract: its file and its `dormant` index key must not
// return, and nothing may still point at it as a live contract.
func TestRunbookOpportunityContractIsRetired(t *testing.T) {
	t.Parallel()

	defaults := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts")
	if _, err := os.Stat(filepath.Join(defaults, "machine", "runbook-opportunity.yaml")); !os.IsNotExist(err) {
		t.Fatalf("machine/runbook-opportunity.yaml must stay retired (err=%v)", err)
	}
	index := readFile(t, filepath.Join(defaults, "index.yaml"))
	for _, forbidden := range []string{"runbook-opportunity", "dormant:"} {
		if strings.Contains(index, forbidden) {
			t.Fatalf("contracts/index.yaml must not mention %q", forbidden)
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
