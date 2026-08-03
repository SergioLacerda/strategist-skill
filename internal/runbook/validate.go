package runbook

import "fmt"

// stringSet builds a lookup set from string values, mirroring
// internal/domain's own stringSet helper. Kept as a package-local copy —
// unexported helpers cannot be shared across packages.
func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

// validateNamedValue mirrors internal/domain's own helper of the same name:
// value is required, and when allowed is non-nil, value must be a member.
func validateNamedValue(token, field, value string, allowed map[string]struct{}) []error {
	if value == "" {
		return []error{fmt.Errorf("%s: %s is required", token, field)}
	}
	if _, ok := allowed[value]; allowed != nil && !ok {
		return []error{fmt.Errorf("%s: %s %q is not an allowed value", token, field, value)}
	}
	return nil
}
