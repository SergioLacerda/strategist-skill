package missionview_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderHumanFullyPopulatedViewIsDeterministic is a golden-style test: it
// asserts RenderHuman's exact output for a fully-populated View so a future
// edit to section/field ordering is a deliberate, visible diff. See
// .analysis/refined/20260921-evaluate-ux-role-confidence-leveling/tasks.md § 3.1.
func TestRenderHumanFullyPopulatedViewIsDeterministic(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{
		{ID: "scout"},
		{ID: "ranger", Slot: "discovery", Phase: 1},
		{ID: "archivist", Slot: "refinement", Phase: 2},
		{ID: "sniper", Slot: "execution", Phase: 4},
	})
	require.NoError(t, err)

	v := missionview.Build(missionview.Input{
		Status:        domain.MissionEngineStatus{MissionID: "m-golden", Phase: domain.PhaseApprovalGate, State: domain.StateApprovalGate},
		Registry:      reg,
		SlotProviders: map[string]string{"discovery": "brainstorming", "refinement": "openspec-propose"},
		Confidence:    telemetry.ConfidenceGateReview{Metrics: telemetry.ConfidenceMetrics{SampleSize: 3}},
		GateOutcome:   "analysis_accepted",
		Levels: []leveling.Record{{
			MissionID: "m-golden",
			Level: leveling.Level{
				Role: "ranger", Model: "Sonnet", Effort: "High", Source: "policy",
				Provider: "anthropic", ModelSource: "policy", EffortSource: "policy",
				Capability: "extended_thinking", FallbackUsed: false,
				PolicyVersion: 3, PolicyDigest: "abc123",
			},
		}},
	})

	var out bytes.Buffer
	require.NoError(t, missionview.RenderHuman(&out, v))

	const want = "Mission\n" +
		"  id: m-golden\n" +
		"  phase: APPROVAL_GATE\n" +
		"  state: APPROVAL_GATE\n" +
		"Journey\n" +
		"  0: scout (role)\n" +
		"  1: ranger (role) provider=brainstorming\n" +
		"  2: archivist (role) provider=openspec-propose\n" +
		"  3: gate (gate)\n" +
		"  4: sniper (role)\n" +
		"Confidence (advisory)\n" +
		"  availability: available\n" +
		"Approval Gate\n" +
		"  availability: available\n" +
		"  outcome: analysis_accepted\n" +
		"LEVELING\n" +
		"  availability: available\n" +
		"  selection: latest_record\n" +
		"  scout: unknown\n" +
		"  ranger: Sonnet-High source=policy model_source=policy effort_source=policy provider=anthropic capability=extended_thinking fallback_used=false fallback_reason= policy_version=3 policy_digest=abc123\n" +
		"  archivist: unknown\n" +
		"  sniper: unknown\n" +
		"Diagnostics\n"

	assert.Equal(t, want, out.String())
}

// TestViewJSONShapeIsStable asserts the strategist-mission-view/v1 JSON
// envelope's top-level keys and section shapes for a populated view and a
// mostly-empty (unavailable/no_sample) view, giving the schema identifier an
// actual compatibility check. See tasks.md § 3.2.
func TestViewJSONShapeIsStable(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		v := missionview.Build(missionview.Input{
			Status:      domain.MissionEngineStatus{MissionID: "m-json", Phase: domain.PhaseRefinement, State: domain.StateRefinement},
			Registry:    domain.DefaultRoleRegistry(),
			GateOutcome: "revision_requested",
			Levels:      []leveling.Record{{MissionID: "m-json", Level: leveling.Level{Role: "ranger", Model: "Sonnet"}}},
		})
		data, err := json.Marshal(v)
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(data, &decoded))

		assert.Equal(t, missionview.SchemaVersion, decoded["schema"])
		assert.Equal(t, "m-json", decoded["mission_id"])
		for _, key := range []string{"lifecycle", "journey", "confidence", "approval_gate", "leveling", "diagnostics"} {
			_, ok := decoded[key]
			assert.Truef(t, ok, "expected top-level key %q in populated view JSON", key)
		}

		gate, ok := decoded["approval_gate"].(map[string]any)
		require.True(t, ok, "approval_gate must decode as an object")
		assert.Equal(t, missionview.Available, gate["availability"])
		assert.Equal(t, "revision_requested", gate["outcome"])
	})

	t.Run("secondary sources unavailable", func(t *testing.T) {
		v := missionview.Build(missionview.Input{
			Status:          domain.MissionEngineStatus{MissionID: "m-empty", Phase: domain.PhaseBootstrap, State: domain.StateInit},
			Registry:        domain.DefaultRoleRegistry(),
			ConfidenceError: assertError("confidence read failed"),
			GateError:       assertError("gate read failed"),
			LevelsError:     assertError("levels read failed"),
		})
		data, err := json.Marshal(v)
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(data, &decoded))

		confidence, ok := decoded["confidence"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, missionview.Unavailable, confidence["availability"])

		gate, ok := decoded["approval_gate"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, missionview.Unavailable, gate["availability"])
		_, hasOutcome := gate["outcome"]
		assert.False(t, hasOutcome, "unavailable gate must not carry a synthesized outcome")

		diagnostics, ok := decoded["diagnostics"].([]any)
		require.True(t, ok)
		assert.Len(t, diagnostics, 3)
	})
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func assertError(msg string) error { return simpleError(msg) }
