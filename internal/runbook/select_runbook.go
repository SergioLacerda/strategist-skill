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

// SelectionPolicy bounds Select's output, per runbook_v2.txt's explicit
// stance against silent, unreasoned runbook application ("eu não aplicaria
// runbooks automaticamente sem explicar a seleção").
type SelectionPolicy struct {
	MaxPrimary    int
	MaxSupporting int
	RequireReason bool
}

// DefaultSelectionPolicy returns runbook_v2.txt's own defaults: at most one
// primary runbook, at most two supporting, and a reason always required.
func DefaultSelectionPolicy() SelectionPolicy {
	return SelectionPolicy{MaxPrimary: 1, MaxSupporting: 2, RequireReason: true}
}

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
func Select(candidates []Runbook, signals MissionSignals, policy SelectionPolicy) ([]Selection, error) {
	if err := validateSelectionInputs(candidates, policy); err != nil {
		return nil, err
	}

	scored := scoreCandidates(candidates, signals)
	sort.SliceStable(scored, func(i, j int) bool {
		return lessScoredCandidate(scored[i], scored[j])
	})

	return buildSelections(scored, policy), nil
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

func buildSelections(scored []scoredCandidate, policy SelectionPolicy) []Selection {
	var selections []Selection
	for _, s := range scored {
		if s.score == 0 {
			continue
		}
		role, ok := nextRole(selections, policy)
		if !ok {
			continue
		}
		selections = append(selections, Selection{
			RunbookID: s.runbook.RunbookID,
			Role:      role,
			Reason:    "matches applies_when: " + strings.Join(s.matched, "; "),
		})
	}
	return selections
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

type scoredCandidate struct {
	runbook Runbook
	score   int
	matched []string
}

func scoreCandidates(candidates []Runbook, signals MissionSignals) []scoredCandidate {
	scored := make([]scoredCandidate, 0, len(candidates))
	for _, rb := range candidates {
		matched := matchAppliesWhen(rb.AppliesWhen, signals)
		scored = append(scored, scoredCandidate{runbook: rb, score: len(matched), matched: matched})
	}
	return scored
}

func matchAppliesWhen(appliesWhen []string, signals MissionSignals) []string {
	var matched []string
	for _, trigger := range appliesWhen {
		if trigger == "" {
			continue
		}
		if triggerMatchesAnySignal(trigger, signals) {
			matched = append(matched, trigger)
		}
	}
	return matched
}

func triggerMatchesAnySignal(trigger string, signals MissionSignals) bool {
	lowerTrigger := strings.ToLower(trigger)
	for _, signal := range signals {
		if signal == "" {
			continue
		}
		if strings.Contains(lowerTrigger, strings.ToLower(signal)) {
			return true
		}
	}
	return false
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
