package handoffconsumer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shadowRig(t *testing.T, body []byte) rig {
	t.Helper()
	return gateRig(t, body, func(p *config.Provider) { p.Shadow = true })
}

func TestAShadowProviderIsCalledAndRecordedButNeverDecides(t *testing.T) {
	r := shadowRig(t, gateAnswers(t, OptionAccept, 0.95))

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusShadow, report.Status)
	assert.True(t, report.WouldApprove)
	assert.Nil(t, report.Delegation, "a shadow reading is never a delegation")
	assert.Empty(t, report.Hints)
	assert.Equal(t, 1, r.sim.Calls())
	line := ledgerLines(t, r.ledger)[0]
	assert.Equal(t, true, line["shadow"])
	assert.Equal(t, true, line["approved"], "the ledger keeps what it would have decided")
}

func TestAShadowSignalIsRecordedWithItsHintsAndHidesThemFromTheAgent(t *testing.T) {
	r := shadowRig(t, answerJSON(t, TransitionArchivistToGate, "jev-1.13.0", OptionReject, 0.2, "validation_stated"))

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusShadow, report.Status)
	assert.False(t, report.WouldApprove)
	assert.Empty(t, report.Hints)
	line := ledgerLines(t, r.ledger)[0]
	assert.Equal(t, []any{"validation_stated"}, line["hints"])
}

func TestAShadowReadingIsNeverReusedAsADelegation(t *testing.T) {
	r := shadowRig(t, gateAnswers(t, OptionAccept, 0.95))
	_ = r.consumer.Precheck(context.Background(), gateRequest())

	sniper := gateRequest()
	sniper.Transition = handoff.TransitionArchivistToSniper
	report, reused := r.consumer.Reuse(sniper)

	require.True(t, reused)
	assert.Equal(t, StatusShadow, report.Status)
	assert.Nil(t, report.Delegation)

	r.consumer.Provider.Shadow = false
	_, reused = r.consumer.Reuse(sniper)
	assert.False(t, reused, "a reading taken in shadow cannot be restated once the provider decides")
}

func TestAShadowReadingDoesNotReachTheConfidenceLedger(t *testing.T) {
	r := shadowRig(t, gateAnswers(t, OptionAccept, 0.95))
	report := r.consumer.Precheck(context.Background(), gateRequest())
	path := filepath.Join(t.TempDir(), "confidence.jsonl")

	require.NoError(t, RecordConfidence(path, gateRequest(), report, "run-1"))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "nothing is written for a shadow reading")
}

func TestSummarizeShadowGroupsByTransitionWithNumbersOnly(t *testing.T) {
	records := []Record{
		{Transition: TransitionArchivistToGate, MissionID: "m1", Shadow: true, Answered: true, Approved: true, Confidence: 0.9, Calibrated: true},
		{Transition: TransitionArchivistToGate, MissionID: "m2", Shadow: true, Answered: true, Confidence: 0.3, Hints: []string{"validation_stated"}},
		{Transition: TransitionArchivistToGate, MissionID: "m2", Shadow: true},
	}

	rows := SummarizeShadow(records)

	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, 3, row.Readings)
	assert.Equal(t, 2, row.Answered)
	assert.Equal(t, 1, row.WouldApprove)
	assert.Equal(t, 1, row.Calibrated)
	assert.Equal(t, 2, row.Missions)
	assert.InDelta(t, 0.6, row.MeanConfident, 1e-9)
	assert.InDelta(t, 0.3, row.MinConfident, 1e-9)
	assert.InDelta(t, 0.9, row.MaxConfident, 1e-9)
	assert.Equal(t, 1, row.Hints["validation_stated"])
}

func TestTheLedgerListsOnlyIntactShadowReadings(t *testing.T) {
	r := shadowRig(t, gateAnswers(t, OptionAccept, 0.95))
	_ = r.consumer.Precheck(context.Background(), gateRequest())
	readings, err := r.consumer.Ledger.Shadow()
	require.NoError(t, err)
	assert.Len(t, readings, 1)

	plain := gateRig(t, gateAnswers(t, OptionAccept, 0.95), nil)
	_ = plain.consumer.Precheck(context.Background(), gateRequest())
	none, err := plain.consumer.Ledger.Shadow()
	require.NoError(t, err)
	assert.Empty(t, none, "a reading of a provider that decides is not a shadow reading")
}
