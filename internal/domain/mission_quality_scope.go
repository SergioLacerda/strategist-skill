package domain

import (
	"fmt"
	"strings"
)

// checkSourceScopeRespected enforces approved source prefixes.
func checkSourceScopeRespected(evidence []Evidence, approvedPrefixes []string) MissionQualityCheckResult {
	if approvedPrefixes == nil {
		return MissionQualityCheckResult{Check: CheckSourceScopeRespected, Applicable: false}
	}
	var violations []string
	for _, e := range evidence {
		if !sourceRefWithinScope(e.SourceRef, approvedPrefixes) {
			violations = append(violations, fmt.Sprintf("evidence %s source_ref %q is outside the approved scope", e.ID, e.SourceRef))
		}
	}
	return MissionQualityCheckResult{
		Check:      CheckSourceScopeRespected,
		Applicable: true,
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

func sourceRefWithinScope(sourceRef string, approvedPrefixes []string) bool {
	for _, prefix := range approvedPrefixes {
		if strings.HasPrefix(sourceRef, prefix) {
			return true
		}
	}
	return false
}
