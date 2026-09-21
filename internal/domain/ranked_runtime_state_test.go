package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRankedRuntimeState_AcceptsCurrentSchema(t *testing.T) {
	raw := []byte(`{"schema_version":"strategist-ranked-runtime/v2","entries":[{"role":"archivist","slot":"refinement","provider":"p","contract_digest":"d","runtime":{"node":"/usr/bin/node","script":"weapon-runtime/p/x.mjs","components":[{"name":"openspec","version":"1.0.0","sha256":"abc"}]}}]}`)

	state, err := ParseRankedRuntimeState(raw)

	require.NoError(t, err)
	entry, ok := state.Entry("refinement", "p")
	require.True(t, ok)
	require.Equal(t, "archivist", entry.Role)
	component, ok := entry.Runtime.Component(RankedRuntimeComponentOpenSpec)
	require.True(t, ok)
	require.Equal(t, "abc", component.SHA256)
}

// A v1 record may carry a private Node (`mode: payload`) or an unverified
// shape; it is never reinterpreted, whatever its runtime fields say.
func TestParseRankedRuntimeState_RejectsLegacySchema(t *testing.T) {
	for name, raw := range map[string]string{
		"v1 payload":   `{"schema_version":"strategist-ranked-runtime/v1","entries":[{"runtime":{"mode":"payload","node":"weapon-runtime/p/node"}}]}`,
		"v1 host node": `{"schema_version":"strategist-ranked-runtime/v1","entries":[{"runtime":{"node":"/usr/bin/node"}}]}`,
		"unversioned":  `{"entries":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseRankedRuntimeState([]byte(raw))

			var legacy *RankedRuntimeStateLegacyError
			require.ErrorAs(t, err, &legacy)
		})
	}
}

func TestParseRankedRuntimeState_RejectsMalformedJSON(t *testing.T) {
	_, err := ParseRankedRuntimeState([]byte("{"))

	require.Error(t, err)
	var legacy *RankedRuntimeStateLegacyError
	require.NotErrorAs(t, err, &legacy)
}

func TestRankedRuntimeStateEntry_MissingReturnsFalse(t *testing.T) {
	_, ok := RankedRuntimeState{}.Entry("refinement", "p")

	require.False(t, ok)
}

func TestRankedRuntimeStateRuntime_NilHasNoComponent(t *testing.T) {
	var runtime *RankedRuntimeStateRuntime

	_, ok := runtime.Component(RankedRuntimeComponentOpenSpec)

	require.False(t, ok)
}
