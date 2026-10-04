package handoffconsumer

import (
	"slices"
	"strings"

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
	var state strings.Builder
	for _, field := range Fields(transition) {
		if !slices.Contains(allowed, field) {
			continue
		}
		if value, ok := source.Field(field); ok && strings.TrimSpace(value) != "" {
			state.WriteString("## " + field + "\n" + strings.TrimSpace(value) + "\n\n")
		}
	}
	switch {
	case state.Len() == 0:
		return "", denied("no allowed field of the artifact can be sent")
	case state.Len() > maxBytes:
		return "", denied("projected state exceeds the data policy size limit")
	}
	return state.String(), nil
}
