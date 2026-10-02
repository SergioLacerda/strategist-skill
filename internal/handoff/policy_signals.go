package handoff

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ResolveArchivistPolicy applies the contract's require/skip predicates and
// precedence to signals extracted from a validated package. It is the only
// production resolver for the Archivist-to-Sniper transition: any matching
// require predicate keeps the challenge required, and only a fully informational
// package skips it. The Approval Gate is never an input.
func ResolveArchivistPolicy(signals RiskSignals) Policy {
	policy := DefaultPolicy()
	policy.Enabled = statusForPolicy(policy, signals) == StatusRequired
	if !policy.Enabled {
		policy.RequiredTypes = nil
	}
	return policy
}

// PolicyIdentity is a stable digest of the policy rules an outcome was decided
// under: the transition, predicates, precedence and challenge shape, but not
// whether this particular package required the challenge.
func PolicyIdentity(policy Policy) string {
	base := DefaultPolicy()
	types := append([]string(nil), base.RequiredTypes...)
	if policy.Transition != TransitionArchivistToSniper {
		types = append([]string(nil), policy.RequiredTypes...)
	}
	sort.Strings(types)
	parts := []string{
		"strategist-handoff-policy/v1", policy.Transition, predicateList(policy.RequireWhen), predicateList(policy.SkipWhen),
		fmt.Sprint(policy.RequirePrecedence), strings.Join(types, ","), fmt.Sprint(policy.RequireAllCritical),
		fmt.Sprint(policy.MaxAttempts), policy.OnFailure,
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
}

func predicateList(predicates []PolicyPredicate) string {
	names := make([]string, len(predicates))
	for i, predicate := range predicates {
		names[i] = string(predicate)
	}
	return strings.Join(names, ",")
}
