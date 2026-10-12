package handoffconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pinnedRecord is a recommended calibration for the gate boundary and the
// resolved model of the simulator.
func pinnedRecord(t *testing.T, threshold float64, mutate func(*calibration.Meta)) calibration.Record {
	t.Helper()
	meta := calibration.Meta{
		Provider: "jev", Model: "jev-1.13.0", Profile: ProfileID, ProfileVersion: ProfileVersion(calibration.BoundaryPackage),
		CorpusDigest: "sha256:c", CatalogVersion: calibration.CatalogVersion(), Now: time.Now(),
	}
	if mutate != nil {
		mutate(&meta)
	}
	record, err := calibration.NewRecord(calibration.Report{Status: calibration.StatusRecommended, Threshold: &threshold, Counts: calibration.Counts{Golden: 60, Defects: 300}}, meta)
	require.NoError(t, err)
	return record
}

func pinned(t *testing.T, record calibration.Record, accept float64) rig {
	t.Helper()
	r := gateRig(t, gateAnswers(t, OptionAccept, accept), func(p *config.Provider) { p.CalibrationID = record.ID })
	r.consumer.Calibrations = func(id string) (calibration.Record, error) {
		if id != record.ID {
			return calibration.Record{}, errors.New("unknown")
		}
		return record, nil
	}
	return r
}

func TestAValidPinnedCalibrationDecidesTheThreshold(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := pinned(t, record, 0.70)

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusApproved, report.Status, "0.70 is above the calibrated 0.60 though below the default 0.80")
	assert.True(t, report.Calibration.Applied)
	assert.InDelta(t, 0.60, report.Threshold, 1e-9)
	assert.Equal(t, 360, report.Calibration.Samples)
	line := ledgerLines(t, r.ledger)[0]
	assert.Equal(t, record.ID, line["calibration_id"])
	assert.Equal(t, true, line["calibrated"])
}

func TestACalibrationCanTightenTheThresholdToo(t *testing.T) {
	r := pinned(t, pinnedRecord(t, 0.95, nil), 0.90)
	assert.Equal(t, StatusSignal, r.consumer.Precheck(context.Background(), gateRequest()).Status, "0.90 exceeds the default 0.80 but does not meet the calibrated 0.95")
}

func TestWithoutAPinTheConfiguredThresholdApplies(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.70), nil)
	report := r.consumer.Precheck(context.Background(), gateRequest())
	assert.Equal(t, StatusSignal, report.Status)
	assert.False(t, report.Calibration.Applied)
	assert.Equal(t, ReasonNonePinned, report.Calibration.Reason)
	assert.InDelta(t, 0.80, report.Threshold, 1e-9)
}

func TestAnInvalidCalibrationFallsBackToTheDefaultWithTheReason(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*calibration.Meta)
		reason string
	}{
		"another model":   {func(m *calibration.Meta) { m.Model = "jev-2.0.0" }, string(calibration.ReasonModelChanged)},
		"another profile": {func(m *calibration.Meta) { m.ProfileVersion = "00000000" }, string(calibration.ReasonProfileChanged)},
		"too old":         {func(m *calibration.Meta) { m.Now = time.Now().Add(-120 * 24 * time.Hour) }, string(calibration.ReasonExpired)},
	} {
		r := pinned(t, pinnedRecord(t, 0.60, tc.mutate), 0.70)
		report := r.consumer.Precheck(context.Background(), gateRequest())
		assert.Equal(t, StatusSignal, report.Status, name)
		assert.False(t, report.Calibration.Applied, name)
		assert.Equal(t, tc.reason, report.Calibration.Reason, name)
		assert.InDelta(t, 0.80, report.Threshold, 1e-9, name)
		assert.Equal(t, tc.reason, ledgerLines(t, r.ledger)[0]["calibration_reason"], name)
	}
}

func TestAMissingOrUnreadableRecordFallsBackToTheDefault(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := pinned(t, record, 0.70)
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return calibration.Record{}, errors.New("gone") }
	report := r.consumer.Precheck(context.Background(), gateRequest())
	assert.Equal(t, StatusSignal, report.Status)
	assert.Equal(t, ReasonRecordUnavailable, report.Calibration.Reason)

	r.consumer.Calibrations = nil
	assert.Equal(t, ReasonRecordUnavailable, r.consumer.Precheck(context.Background(), gateRequest()).Calibration.Reason)
}

func TestANotRecommendedOrTamperedCalibrationIsNeverApplied(t *testing.T) {
	record, err := calibration.NewRecord(calibration.Report{Status: calibration.StatusNotFit, FailingGate: "separation"},
		calibration.Meta{Provider: "jev", Model: "jev-1.13.0", Profile: ProfileID, ProfileVersion: ProfileVersion(calibration.BoundaryPackage), Now: time.Now()})
	require.NoError(t, err)
	r := pinned(t, record, 0.95)
	assert.Equal(t, string(calibration.ReasonNotRecommended), r.consumer.Precheck(context.Background(), gateRequest()).Calibration.Reason)

	good := pinnedRecord(t, 0.60, nil)
	edited := 0.10
	good.Threshold = &edited
	r = pinned(t, good, 0.70)
	assert.Equal(t, string(calibration.ReasonInvalid), r.consumer.Precheck(context.Background(), gateRequest()).Calibration.Reason)
}

func TestTheDelegationNamesTheCalibrationThatDecidedIt(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := pinned(t, record, 0.70)
	report := r.consumer.Precheck(context.Background(), Request{MissionID: "m-1", Transition: handoff.TransitionArchivistToSniper, Subject: "sha256:package", Source: gatePackageSource()})
	require.Equal(t, StatusApproved, report.Status)
	require.NoError(t, report.Delegation.Validate())
	assert.Equal(t, record.ID, report.Delegation.CalibrationID)
}

func TestTheSniperEvaluationReusesTheGateReadingWithTheCurrentCalibration(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := pinned(t, record, 0.70)
	gate := r.consumer.Precheck(context.Background(), gateRequest())
	require.Equal(t, StatusApproved, gate.Status)

	reused, ok := r.consumer.Reuse(Request{MissionID: "m-1", Transition: handoff.TransitionArchivistToSniper, Subject: "sha256:package", Source: gatePackageSource()})
	require.True(t, ok)
	assert.Equal(t, StatusApproved, reused.Status)
	assert.Equal(t, record.ID, reused.Delegation.CalibrationID)

	r.consumer.Calibrations = func(string) (calibration.Record, error) { return calibration.Record{}, errors.New("gone") }
	reused, ok = r.consumer.Reuse(Request{MissionID: "m-1", Transition: handoff.TransitionArchivistToSniper, Subject: "sha256:package", Source: gatePackageSource()})
	require.True(t, ok)
	assert.Equal(t, StatusSignal, reused.Status, "the calibration is gone: the default threshold rejects 0.70")
	assert.Nil(t, reused.Delegation)
}

func TestTheConfidenceClaimCitesTheCalibrationOnlyWhenItWasApplied(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	applied := providerClaimFor(t, pinned(t, record, 0.70))
	assert.Equal(t, domain.CalibrationObserved, applied.CalibrationStatus)
	assert.Equal(t, 360, applied.SampleSize)
	assert.Contains(t, applied.EvidenceIDs, "provider-calibration-"+record.ID)
	assert.Contains(t, applied.Statement, record.ID)

	plain := providerClaimFor(t, gateRig(t, gateAnswers(t, OptionAccept, 0.70), nil))
	assert.Equal(t, domain.CalibrationNoSample, plain.CalibrationStatus)
	assert.Zero(t, plain.SampleSize)
	assert.Contains(t, plain.Statement, "uncalibrated")
}

func providerClaimFor(t *testing.T, r rig) domain.ConfidenceClaim {
	t.Helper()
	request := gateRequest()
	report := r.consumer.Precheck(context.Background(), request)
	claim, evidence := providerClaim(request, report, "provider-conformance-"+request.Transition)
	require.NoError(t, domain.ValidateConfidenceClaim(claim, evidence))
	return claim
}

func TestRecordConfidenceWritesTheCalibratedClaim(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := pinned(t, record, 0.70)
	request := gateRequest()
	report := r.consumer.Precheck(context.Background(), request)
	path := filepath.Join(t.TempDir(), "confidence-records.jsonl")
	require.NoError(t, RecordConfidence(path, request, report, "handoff-archivist_to_gate"))
	assert.FileExists(t, path)
}

// noProbabilities is a documented-shape answer that reports only the confidence of the choice.
func noProbabilities(t *testing.T, confidence float64) []byte {
	t.Helper()
	answers := map[string]any{AggregateKey: map[string]any{"type": "choice", "choice": OptionAccept, "confidence": confidence}}
	for _, c := range Criteria(TransitionArchivistToGate) {
		answers[c.Key] = map[string]any{"type": "noul", "noul": 1.0}
	}
	raw, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": answers, "usage": map[string]int{"input_tokens": 100, "output_tokens": 10}})
	require.NoError(t, err)
	return raw
}

func TestACalibrationNeverAppliesToAnAnswerWithoutAProbabilityOfAccept(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil)
	r := gateRig(t, noProbabilities(t, 0.70), func(p *config.Provider) { p.CalibrationID = record.ID })
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return record, nil }

	report := r.consumer.Precheck(context.Background(), gateRequest())

	assert.False(t, report.Calibration.Applied, "a threshold on the probability of accept is not a threshold on the confidence")
	assert.Equal(t, ReasonProbabilityUnavailable, report.Calibration.Reason)
	assert.InDelta(t, 0.80, report.Threshold, 1e-9)
	assert.Equal(t, StatusSignal, report.Status)
}

func TestTheJudgeMarksAnAnswerWithoutAProbabilityUnusableForCalibration(t *testing.T) {
	r := gateRig(t, noProbabilities(t, 0.95), nil)
	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, StateMissingProbability, verdict.State)
	assert.Empty(t, verdict.Decision)
}
