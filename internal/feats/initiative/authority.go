package initiative

import (
	"fmt"
	"strings"
)

var initiativeOwnedAuthorities = []string{"advice", "diligence", "alignment"}

var initiativeForbiddenAuthorities = []string{
	"model", "provider", "capability", "effort", "level_source",
	"role_binding", "weapon_binding", "stage", "route", "pipeline_bypass",
	"approval_gate", "implementation_authorization",
}

// DefaultAuthority returns the explicit authority boundary for the INITIATIVE
// Feat. The slices are copied so callers cannot mutate the policy contract.
func DefaultAuthority() PolicyAuthority {
	return PolicyAuthority{
		Owns:       append([]string(nil), initiativeOwnedAuthorities...),
		DoesNotOwn: append([]string(nil), initiativeForbiddenAuthorities...),
	}
}

func validateAuthorityBoundary(authority PolicyAuthority) error {
	owned, err := normalizeAuthoritySet("owns", authority.Owns)
	if err != nil {
		return err
	}
	forbidden, err := normalizeAuthoritySet("does_not_own", authority.DoesNotOwn)
	if err != nil {
		return err
	}
	if err := requireAuthorities("owns", owned, initiativeOwnedAuthorities); err != nil {
		return err
	}
	if err := requireAuthorities("does_not_own", forbidden, initiativeForbiddenAuthorities); err != nil {
		return err
	}
	return rejectAuthorityOverlap(owned, forbidden)
}

func requireAuthorities(field string, actual map[string]bool, required []string) error {
	for _, name := range required {
		if !actual[name] {
			return fmt.Errorf("initiative_policy_invalid: authority.%s must include %q", field, name)
		}
	}
	return nil
}

func rejectAuthorityOverlap(owned, forbidden map[string]bool) error {
	for name := range owned {
		if forbidden[name] {
			return fmt.Errorf("initiative_policy_invalid: authority %q cannot be both owned and forbidden", name)
		}
	}
	return nil
}

func normalizeAuthoritySet(field string, values []string) (map[string]bool, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("initiative_policy_invalid: authority.%s is required", field)
	}
	set := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return nil, fmt.Errorf("initiative_policy_invalid: authority.%s contains an empty value", field)
		}
		if set[value] {
			return nil, fmt.Errorf("initiative_policy_invalid: authority.%s contains duplicate %q", field, value)
		}
		set[value] = true
	}
	return set, nil
}
