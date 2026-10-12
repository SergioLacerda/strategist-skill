package handoffconsumer

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pairAnswer(t *testing.T, kind, obligation, verification float64) []byte {
	t.Helper()
	answers := map[string]any{
		calibration.DimKind:         map[string]any{"type": "noul", "noul": kind},
		calibration.DimObligation:   map[string]any{"type": "noul", "noul": obligation},
		calibration.DimVerification: map[string]any{"type": "noul", "noul": verification},
	}
	raw, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": answers, "usage": map[string]int{"input_tokens": 180, "output_tokens": 12}})
	require.NoError(t, err)
	return raw
}

func pairRig(t *testing.T, body []byte) rig {
	t.Helper()
	return newRig(t, jevtest.Reply(body), nil)
}

func judgeOfPairs(r rig) PairJudge {
	return PairJudge{Mechanism: r.consumer.Mechanism, Provider: r.consumer.Provider, Name: "jev", Version: "1"}
}

func TestThePairJudgeReturnsOneScorePerDimensionAndTheUsage(t *testing.T) {
	r := pairRig(t, pairAnswer(t, 0.92, 0.15, 0.80))
	pair := calibration.SyntheticPairs(1)[0]

	v := judgeOfPairs(r).JudgePair(context.Background(), pair)

	require.Empty(t, v.State)
	assert.InDelta(t, 0.92, v.Scores[calibration.DimKind], 1e-9)
	assert.InDelta(t, 0.15, v.Scores[calibration.DimObligation], 1e-9)
	assert.InDelta(t, 0.80, v.Scores[calibration.DimVerification], 1e-9)
	assert.Equal(t, "jev-1.13.0", v.Model)
	assert.Equal(t, 180, v.InputTokens)
}

func TestThePairRequestCarriesTheStateAndTheThreeQuestionsButNeverTheLabel(t *testing.T) {
	r := pairRig(t, pairAnswer(t, 1, 1, 1))
	pair := calibration.SyntheticPairs(1)[0]
	_ = judgeOfPairs(r).JudgePair(context.Background(), pair)
	body := r.sim.LastBody()
	assert.Contains(t, body, "target.declared_kind")
	assert.Contains(t, body, "requirement.text")
	assert.Contains(t, body, "declared_kind: implementation")
	for _, secret := range []string{pair.Family, pair.ID, pair.Parent, "oracle"} {
		assert.NotContains(t, body, secret)
	}
	assert.Len(t, PairQuestions(), len(calibration.PairDimensions()))
}

func TestAMissingAnswerOrAnOversizedStateIsATypedFailure(t *testing.T) {
	partial, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": map[string]any{calibration.DimKind: map[string]any{"type": "noul", "noul": 1.0}}, "usage": map[string]int{}})
	require.NoError(t, err)
	r := pairRig(t, partial)
	assert.Equal(t, "invalid_response", judgeOfPairs(r).JudgePair(context.Background(), calibration.SyntheticPairs(1)[0]).State)

	small := newRig(t, jevtest.Reply(pairAnswer(t, 1, 1, 1)), func(p *config.Provider) { p.DataPolicy.MaxStateBytes = 20 })
	assert.Equal(t, "data_policy_denied", judgeOfPairs(small).JudgePair(context.Background(), calibration.SyntheticPairs(1)[0]).State)

	down := newRig(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, nil)
	assert.NotEmpty(t, judgeOfPairs(down).JudgePair(context.Background(), calibration.SyntheticPairs(1)[0]).State)
}

func TestThePairProfileHasItsOwnStableVersion(t *testing.T) {
	first := PairProfileVersion()
	assert.Equal(t, first, PairProfileVersion())
	assert.Len(t, PairProfileVersion(), 8)
	assert.NotEqual(t, PairProfileVersion(), ProfileVersion(calibration.BoundaryPackage))
}
