package runbook

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MissionSignals is the set of free-text signals observed for the current
// mission (e.g. symptom keywords, phase names) that a candidate runbook's
// AppliesWhen entries are matched against.
type MissionSignals []string

// SelectionRole names whether a Selection is the mission's primary runbook
// or one of its supporting runbooks.
type SelectionRole string

// Selection roles a candidate can be assigned.
const (
	RolePrimary    SelectionRole = "primary"
	RoleSupporting SelectionRole = "supporting"
)

// Selection is one candidate Select chose, with the reason it matched.
type Selection struct {
	RunbookID string
	Role      SelectionRole
	Reason    string
}

// Select scores candidates by how many of their AppliesWhen entries match
// signals (case-insensitive substring match), then assigns the
// highest-scoring candidates as primary and the next as supporting, bounded
// by policy. A candidate with zero matches is never selected — Select never
// applies a runbook without a matched trigger to point to. When
// policy.RequireReason is true (the default), every returned Selection
// carries a non-empty Reason naming the signals it matched; this holds
// unconditionally by construction, since only matched candidates are ever
// selected.
//
// Before scoring, candidates that fail policy's staleness (MaxAge), trust
// (MinTrust), or explicit-conflict (ConflictsWithHigherTrust) checks are
// rejected outright — they never compete for a primary/supporting slot at
// all, regardless of how well their applies_when text would have matched.
// rejections carries one Rejection per candidate that did not end up in
// selections, covering every rejection path (no match, stale, trust,
// conflict, budget, or policy cap) — see design.md item 4's auditability
// requirement. Policy fields (SelectionPolicy, Rejection, and the trust/
// staleness helpers they use) live in select_runbook_policy.go.
func Select(candidates []Runbook, signals MissionSignals, policy SelectionPolicy) (selections []Selection, rejections []Rejection, err error) {
	if err := validateSelectionInputs(candidates, policy); err != nil {
		return nil, nil, err
	}

	var eligible []Runbook
	for _, rb := range candidates {
		switch {
		case rb.ConflictsWithHigherTrust:
			rejections = append(rejections, Rejection{RunbookID: rb.RunbookID, Reason: RejectionConflictsHigherTrust})
		case !meetsMinTrust(rb.Trust, policy.MinTrust):
			rejections = append(rejections, Rejection{RunbookID: rb.RunbookID, Reason: RejectionBelowMinTrust})
		case isStale(rb, policy):
			rejections = append(rejections, Rejection{RunbookID: rb.RunbookID, Reason: RejectionStale})
		default:
			eligible = append(eligible, rb)
		}
	}

	scored := scoreCandidates(eligible, signals)
	sort.SliceStable(scored, func(i, j int) bool {
		return lessScoredCandidate(scored[i], scored[j])
	})

	built, rejectedScored := buildSelections(scored, policy)
	selections = built
	rejections = append(rejections, rejectedScored...)
	return selections, rejections, nil
}

func validateSelectionInputs(candidates []Runbook, policy SelectionPolicy) error {
	if policy.MaxPrimary < 0 || policy.MaxSupporting < 0 {
		return errors.New("selection_policy_invalid: max_primary and max_supporting must be non-negative")
	}
	return validateNoDuplicateIDs(candidates)
}

func lessScoredCandidate(a, b scoredCandidate) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	return a.runbook.RunbookID < b.runbook.RunbookID
}

// buildSelections walks scored (already sorted highest-first) and assigns
// each candidate a role, a budget outcome, or a rejection reason. spent
// tracks the running EstimatedTokens total against policy.TokenBudget; an
// over-budget candidate is rejected but does not stop the walk — a smaller,
// lower-ranked candidate later in scored may still fit the remaining budget.
func buildSelections(scored []scoredCandidate, policy SelectionPolicy) (selections []Selection, rejections []Rejection) {
	spent := 0
	for _, s := range scored {
		if s.score == 0 {
			rejections = append(rejections, Rejection{RunbookID: s.runbook.RunbookID, Reason: RejectionNoMatch})
			continue
		}
		role, ok := nextRole(selections, policy)
		if !ok {
			rejections = append(rejections, Rejection{RunbookID: s.runbook.RunbookID, Reason: RejectionPolicyCapReached})
			continue
		}
		if policy.TokenBudget > 0 && spent+s.runbook.EstimatedTokens > policy.TokenBudget {
			rejections = append(rejections, Rejection{RunbookID: s.runbook.RunbookID, Reason: RejectionOverBudget})
			continue
		}
		spent += s.runbook.EstimatedTokens
		selections = append(selections, Selection{
			RunbookID: s.runbook.RunbookID,
			Role:      role,
			Reason:    "matches applies_when: " + strings.Join(s.matched, "; "),
		})
	}
	return selections, rejections
}

func nextRole(selections []Selection, policy SelectionPolicy) (SelectionRole, bool) {
	if len(selections) < policy.MaxPrimary {
		return RolePrimary, true
	}
	if countByRole(selections, RoleSupporting) >= policy.MaxSupporting {
		return "", false
	}
	return RoleSupporting, true
}

func countByRole(selections []Selection, role SelectionRole) int {
	n := 0
	for _, s := range selections {
		if s.Role == role {
			n++
		}
	}
	return n
}

func validateNoDuplicateIDs(candidates []Runbook) error {
	seen := make(map[string]struct{}, len(candidates))
	for _, rb := range candidates {
		if _, ok := seen[rb.RunbookID]; ok {
			return fmt.Errorf("selection_candidates_invalid: duplicate runbook_id %q", rb.RunbookID)
		}
		seen[rb.RunbookID] = struct{}{}
	}
	return nil
}
