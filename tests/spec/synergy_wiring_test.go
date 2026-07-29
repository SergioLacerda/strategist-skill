//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// W8 (deep analysis 2026-07-26, proposals P5 + P2): Critic-at-Gate and Riposte
// wiring. P5 surfaces the response-critic result at the Approval Gate — the one
// human decision point (fixes Y3). P2 turns gate declines/revisions and
// unselected side quests (sq_backlog, Y2) into offered backlog captures via
// Riposte's own normalize+capture machinery (migrated off quick-draw.yaml by
// remove-quick-draw-cli-skill-residual T1, 2026-07-28).

func gateContractPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "approval-gate.yaml")
}

func ripostePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "riposte.yaml")
}

// P5: the gate machine contract must declare the critic display — score shown,
// `review` pre-suggested on critic fail, and an explicit no-auto-reject invariant.
func TestApprovalGateDeclaresCriticDisplay(t *testing.T) {
	t.Parallel()

	content := readFile(t, gateContractPath(t))
	for _, needle := range []string{
		"critic_display:",
		"on_fail_suggest: review",
		"no_rubric",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("approval-gate.yaml missing critic display declaration %q", needle)
		}
	}
	if !strings.Contains(content, "critic result never auto-rejects") {
		t.Errorf("approval-gate.yaml missing the critic no-auto-reject invariant")
	}
}

// P2: decline and revision outcomes must offer a Riposte capture (never silent,
// never automatic).
func TestApprovalGateOutcomesOfferRiposte(t *testing.T) {
	t.Parallel()

	content := readFile(t, gateContractPath(t))
	if strings.Count(content, "riposte_offer: true") < 2 {
		t.Errorf("approval-gate.yaml must declare riposte_offer: true on both on_revision_requested and on_reject")
	}
}

// P2: the riposte machine contract exists, owns its own normalize+capture
// machinery, covers the three triggers, and carries the doctrine invariants.
func TestRiposteContractShape(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(ripostePath(t))
	if err != nil {
		t.Fatalf("read riposte.yaml: %v", err)
	}
	var doc struct {
		Module string `yaml:"module"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse riposte.yaml: %v", err)
	}
	if doc.Module != "riposte" {
		t.Fatalf("riposte.yaml module = %q, want %q", doc.Module, "riposte")
	}

	content := string(data)
	for _, needle := range []string{
		"gate_revision_requested",
		"gate_rejected",
		"mission_close_sq_backlog",
		"riposte_normalize",
		"riposte_capture",
		"origin: riposte",
		"never write without an explicit user confirmation",
		"never converts a decline into a new mission automatically",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("riposte.yaml missing %q", needle)
		}
	}
	if strings.Contains(content, "quick-draw.yaml#phases") {
		t.Errorf("riposte.yaml should no longer depend on quick-draw.yaml's phases — it owns its own normalize+capture machinery")
	}
}

// Riposte must be registered in the authoritative manifest for the gate phase
// (mission-close drain rides the response phase).
func TestRiposteRegisteredInContractIndex(t *testing.T) {
	t.Parallel()

	index := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "index.yaml"))
	if !strings.Contains(index, "machine/riposte.yaml") {
		t.Errorf("contracts/index.yaml does not register machine/riposte.yaml")
	}
}

// P5+P2 narrative: the gate display shows the critic line and the narrative
// documents the Riposte offer; sq_backlog gains its drain path.
func TestGateNarrativeShowsCriticAndRiposte(t *testing.T) {
	t.Parallel()

	content := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "05-approval-gate.md"))
	for _, needle := range []string{
		"🎯 CRITIC",
		"## Riposte",
		"sq_backlog",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("05-approval-gate.md missing %q", needle)
		}
	}
}

// W10 (deep analysis 2026-07-26, proposals P3 + P6): Keen Senses radar and
// Opportunity Attack jewel harvesting.

// P3: the keen-senses machine contract exists, is registered for bootstrap, and
// carries the surface-only doctrine (never archives, never deprecates).
func TestKeenSensesContractShape(t *testing.T) {
	t.Parallel()

	path := filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "keen-senses.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read keen-senses.yaml: %v", err)
	}
	var doc struct {
		Module string `yaml:"module"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse keen-senses.yaml: %v", err)
	}
	if doc.Module != "keen-senses" {
		t.Fatalf("keen-senses.yaml module = %q, want %q", doc.Module, "keen-senses")
	}

	content := string(data)
	for _, needle := range []string{
		"stale_captured_entries",
		"stale_jewels",
		"stale_chests",
		"never archives",
		"never deprecates",
		"internal/stale",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("keen-senses.yaml missing %q", needle)
		}
	}

	index := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "index.yaml"))
	if !strings.Contains(index, "machine/keen-senses.yaml") {
		t.Errorf("contracts/index.yaml does not register machine/keen-senses.yaml")
	}
}

// P6: Opportunity Attack harvests jewel candidates on criteria hit — always
// status proposed, never bypassing the mine curation gates.
func TestOpportunityAttackHarvestsJewelCandidates(t *testing.T) {
	t.Parallel()

	content := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "opportunity-attack.yaml"))
	for _, needle := range []string{
		"jewel_candidate",
		"status: proposed",
		"mine curation gates",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("opportunity-attack.yaml missing jewel harvesting declaration %q", needle)
		}
	}
}
