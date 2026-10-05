package gatecriteria

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/require"
)

func publishedCalibration(t *testing.T, f fixture) calibration.Record {
	t.Helper()
	threshold := 0.84
	held := calibration.SetResult{Positives: 20, Defects: 70}
	record, err := calibration.NewRecord(
		calibration.Report{Status: calibration.StatusRecommended, Threshold: &threshold, HeldOut: &held, Counts: calibration.Counts{Golden: 60, Defects: 300}},
		calibration.Meta{Provider: "jev", Model: "jev-1.13.0", Profile: handoffconsumer.ProfileID, ProfileVersion: "abcd1234", Now: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.NoError(t, dojo.SaveCalibration(f.basePath, record))
	return record
}

func TestACalibratedReadingShowsTheEvidenceOfItsCalibration(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	record := publishedCalibration(t, f)
	f.ledger(t, handoffconsumer.Record{Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Provider: "jev", Attempted: true, Answered: true,
		Confidence: 0.90, Threshold: 0.84, CalibrationID: record.ID, Calibrated: true})

	provider := row(t, Build(f.input()), "provider.ranger_to_archivist")

	require.Equal(t, StateEvaluated, provider.State)
	require.Equal(t, "JEV confiança: 90% (limiar 84%) — aprovado, calibrado (modelo jev-1.13.0, perfil handoff-v1, 60 casos golden, falsa aprovação held-out 0.0%, 2026-10-04)", provider.Value)
}

func TestAPinnedButUnusableCalibrationIsSaidToBeNotCalibratedWithTheReason(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	f.ledger(t, handoffconsumer.Record{Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Provider: "jev", Attempted: true, Answered: true,
		Confidence: 0.90, Threshold: 0.80, CalibrationID: "cal-20261004-aaaaaaaa", CalibrationReason: "model_changed"})

	provider := row(t, Build(f.input()), "provider.ranger_to_archivist")

	require.Equal(t, "JEV confiança: 90% (limiar 80%) — aprovado, não calibrado (model_changed)", provider.Value)
}

func TestACalibratedReadingWhoseRecordIsGoneStillSaysCalibratedWithTheID(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	f.ledger(t, handoffconsumer.Record{Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Provider: "jev", Attempted: true, Answered: true,
		Confidence: 0.90, Threshold: 0.84, CalibrationID: "cal-20261004-bbbbbbbb", Calibrated: true})
	require.Contains(t, row(t, Build(f.input()), "provider.ranger_to_archivist").Value, "calibrado (cal-20261004-bbbbbbbb)")
}

func TestADelegationNamesItsCalibration(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	record := publishedCalibration(t, f)
	delegation := &handoff.Delegation{Provider: "jev", Model: "jev-1.13.0", BindingDigest: "sha256:b", Capability: "handoff.validate", Criterion: "handoff_acceptable",
		Subject: "sha256:artifact", Checks: handoff.DelegableChecks(), Confidence: 0.9, Threshold: 0.84, CalibrationID: record.ID}
	f.outcome(t, handoff.TransitionRangerToArchivist, handoff.OutcomePassed, delegation)
	require.Contains(t, row(t, Build(f.input()), "provider.ranger_to_archivist").Value, "calibrado (modelo jev-1.13.0")
}
