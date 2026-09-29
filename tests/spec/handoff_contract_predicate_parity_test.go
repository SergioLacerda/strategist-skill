//go:build spec

package spec_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"gopkg.in/yaml.v3"
)

// TestHandoffContractPredicateParity guards against handoff-contract.yaml's
// documented require_when/skip_when lists for the Archivist->Sniper
// (archivist_to_sniper) transition drifting away from what
// internal/handoff.DefaultPolicy() actually evaluates. The contract is a
// human-maintained document — nothing parses it into the running policy — so
// a one-sided edit on either side previously went uncaught (see
// .analysis/archived/20260928-avaliar-pending-items-relevance-adr.md and its
// sibling analysis for the drift this test was written to prevent finding
// again by hand).
func TestHandoffContractPredicateParity(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	raw := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "machine", "handoff-contract.yaml"))

	var doc struct {
		HandoffVerificationPolicy struct {
			RequireWhen []string `yaml:"require_when"`
			SkipWhen    []string `yaml:"skip_when"`
		} `yaml:"handoff_verification_policy"`
	}
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("parse handoff-contract.yaml: %v", err)
	}

	policy := handoff.DefaultPolicy()

	assertSameSet(t, "require_when", doc.HandoffVerificationPolicy.RequireWhen, predicateStrings(policy.RequireWhen))
	assertSameSet(t, "skip_when", doc.HandoffVerificationPolicy.SkipWhen, predicateStrings(policy.SkipWhen))
}

func predicateStrings(predicates []handoff.PolicyPredicate) []string {
	out := make([]string, len(predicates))
	for i, p := range predicates {
		out[i] = string(p)
	}
	return out
}

// assertSameSet fails with the specific missing/extra entries on either side,
// so a future drift names exactly what to fix instead of just "mismatch".
func assertSameSet(t *testing.T, listName string, contractSide, codeSide []string) {
	t.Helper()

	inContractNotCode := diff(contractSide, codeSide)
	inCodeNotContract := diff(codeSide, contractSide)

	if len(inContractNotCode) > 0 {
		t.Errorf("handoff-contract.yaml's %s documents %v, which DefaultPolicy() does not evaluate for archivist_to_sniper — remove it from the contract or add it as a PolicyPredicate the policy references", listName, inContractNotCode)
	}
	if len(inCodeNotContract) > 0 {
		t.Errorf("DefaultPolicy()'s %s evaluates %v, which handoff-contract.yaml does not document — add it to the contract's %s list", listName, inCodeNotContract, listName)
	}
}

// diff returns the sorted elements of a not present in b.
func diff(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, v := range b {
		inB[v] = true
	}
	var out []string
	for _, v := range a {
		if !inB[v] {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
