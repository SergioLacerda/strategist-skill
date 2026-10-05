package handoffconsumer

import (
	"slices"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Source supplies the text of a contract field of the artifact under check.
type Source interface {
	Field(name string) (string, bool)
}

// MapSource is an in-memory Source.
type MapSource map[string]string

// Field returns the value of a field.
func (m MapSource) Field(name string) (string, bool) {
	value, ok := m[name]
	return value, ok
}

func denied(reason string) error {
	return integration.NewError(integration.StateDataPolicyDenied, integration.Correlation{Provider: "jev", Consumer: "handoff"}, reason)
}

// Project builds the state text sent to the provider. A field reaches it only if
// the catalog reads it, the operator's data policy allows it and the artifact
// has it; anything else, including a field the policy lists that the catalog does
// not know, never leaves. An empty or oversized projection is denied rather than
// truncated, so the provider never sees a partial artifact presented as whole.
func Project(transition string, source Source, allowed []string, maxBytes int) (string, error) {
	return ProjectFor(calibration.ProfileV1, transition, source, allowed, maxBytes)
}

// ProjectFor projects under a profile. Profile v2 shows an allowed catalog field the
// artifact lacks as `(absent)` (no content, so nothing leaves that the policy did not
// allow); an artifact with no allowed field at all is still denied.
func ProjectFor(profile, transition string, source Source, allowed []string, maxBytes int) (string, error) {
	var state strings.Builder
	present := 0
	for _, field := range Fields(transition) {
		if !slices.Contains(allowed, field) {
			continue
		}
		block, has := fieldBlock(profile, field, source)
		state.WriteString(block)
		if has {
			present++
		}
	}
	switch {
	case present == 0:
		return "", denied("no allowed field of the artifact can be sent")
	case state.Len() > maxBytes:
		return "", denied("projected state exceeds the data policy size limit")
	}
	return state.String(), nil
}

// fieldBlock is the text of one allowed field: its content when the artifact has it,
// the (absent) marker under profile v2 when it does not, and nothing otherwise.
func fieldBlock(profile, field string, source Source) (string, bool) {
	if value, ok := source.Field(field); ok && strings.TrimSpace(value) != "" {
		return "## " + field + "\n" + strings.TrimSpace(value) + "\n\n", true
	}
	if calibration.ShowsAbsent(calibration.ResolveProfile(profile)) {
		return "## " + field + "\n(absent)\n\n", false
	}
	return "", false
}
