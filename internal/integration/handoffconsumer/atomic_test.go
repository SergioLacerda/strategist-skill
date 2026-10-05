package handoffconsumer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func atomicAnswer(t *testing.T, scores map[string]float64) []byte {
	t.Helper()
	answers := map[string]any{}
	for key, score := range scores {
		answers[key] = map[string]any{"type": "noul", "noul": score}
	}
	raw, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": answers, "usage": map[string]int{"input_tokens": 700, "output_tokens": 40}})
	require.NoError(t, err)
	return raw
}

func allCriteria(score float64) map[string]float64 {
	scores := map[string]float64{}
	for _, criterion := range Criteria(TransitionArchivistToGate) {
		scores[criterion.Key] = score
	}
	return scores
}

func atomicJudge(r rig) Judge {
	judge := judgeOf(r)
	judge.Profile = calibration.ProfileAtomicV1
	return judge
}

func TestAtomicQuestionsAreOneNoulPerCriterionWithNoAggregate(t *testing.T) {
	questions := QuestionsFor(calibration.ProfileAtomicV1, TransitionArchivistToGate)

	require.Len(t, questions, len(Criteria(TransitionArchivistToGate)))
	for _, q := range questions {
		assert.Equal(t, integration.KindNoul, q.Kind)
		assert.NotEqual(t, AggregateKey, q.Key)
		assert.Contains(t, q.Instructions, "About the field `")
		assert.Contains(t, q.Instructions, "(absent)")
	}
	assert.Nil(t, QuestionsFor(calibration.ProfileAtomicV1, "unknown"))
}

func TestTheNegativeCriterionIsAskedPositivelyInTheAtomicProfile(t *testing.T) {
	var asked string
	for _, q := range QuestionsFor(calibration.ProfileAtomicV1, TransitionArchivistToGate) {
		if q.Key == "implementation_not_materialized" {
			asked = q.Instructions
		}
	}
	assert.Contains(t, asked, "declared type matches the work its action describes")
	assert.NotContains(t, asked, "No implementation_handoff task")
	for _, c := range Criteria(TransitionArchivistToGate) {
		if c.Key == "implementation_not_materialized" {
			continue
		}
		assert.Contains(t, atomicQuestionText(t, c.Key), c.Instructions, "a criterion with no override keeps its catalog wording")
	}
}

func atomicQuestionText(t *testing.T, key string) string {
	t.Helper()
	for _, q := range QuestionsFor(calibration.ProfileAtomicV1, TransitionArchivistToGate) {
		if q.Key == key {
			return q.Instructions
		}
	}
	require.FailNow(t, "no such question", key)
	return ""
}

func TestAtomicProfileVersionDiffersFromTheOthersAndIsStable(t *testing.T) {
	atomic := ProfileVersionOf(calibration.ProfileAtomicV1, calibration.BoundaryPackage)
	assert.Equal(t, atomic, ProfileVersionOf(calibration.ProfileAtomicV1, calibration.BoundaryPackage))
	assert.NotEqual(t, atomic, ProfileVersionOf(calibration.ProfileV2, calibration.BoundaryPackage))
	assert.NotEqual(t, atomic, ProfileVersionOf(calibration.ProfileV1, calibration.BoundaryPackage))
}

func TestAtomicProfileShowsAnAbsentFieldLikeV2(t *testing.T) {
	allowed := Fields(TransitionArchivistToGate)
	text, err := ProjectFor(calibration.ProfileAtomicV1, TransitionArchivistToGate, MapSource{"approved_scope": "- allowed: docs/**"}, allowed, 4096)
	require.NoError(t, err)
	assert.Contains(t, text, "## acceptance_checks\n(absent)")
}

func TestTheAtomicJudgeReportsTheLowestCriterionAsTheProbabilityOfAccept(t *testing.T) {
	scores := allCriteria(0.9)
	scores["validation_stated"] = 0.31
	r := newRig(t, jevtest.Reply(atomicAnswer(t, scores)), allFields)

	verdict := atomicJudge(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)

	require.Empty(t, verdict.State)
	assert.InDelta(t, 0.31, verdict.PAccept, 1e-9)
	assert.Equal(t, OptionReject, verdict.Decision)
	assert.InDelta(t, 0.69, verdict.Probabilities[OptionReject], 1e-9)
	assert.InDelta(t, 0.31, verdict.Scores["validation_stated"], 1e-9)
	assert.Len(t, verdict.Scores, len(Criteria(TransitionArchivistToGate)))
	assert.Equal(t, 700, verdict.InputTokens)
}

func TestTheAtomicJudgeAcceptsWhenEveryCriterionHolds(t *testing.T) {
	r := newRig(t, jevtest.Reply(atomicAnswer(t, allCriteria(0.8))), allFields)
	verdict := atomicJudge(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, OptionAccept, verdict.Decision)
	assert.InDelta(t, 0.8, verdict.PAccept, 1e-9)
}

func TestTheAtomicJudgeRefusesAnAnswerMissingACriterion(t *testing.T) {
	scores := allCriteria(0.9)
	delete(scores, "scope_bounded")
	r := newRig(t, jevtest.Reply(atomicAnswer(t, scores)), allFields)
	verdict := atomicJudge(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, string(integration.StateInvalidResponse), verdict.State)
	assert.Empty(t, verdict.Decision)
}

func TestTheAtomicRequestAsksOnlyTheCriterionQuestions(t *testing.T) {
	r := newRig(t, jevtest.Reply(atomicAnswer(t, allCriteria(0.9))), allFields)
	_ = atomicJudge(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	body := r.sim.LastBody()
	assert.Contains(t, body, "scope_bounded")
	assert.NotContains(t, body, AggregateKey)
}

func TestTheAtomicJudgeMapsAProviderFailureToAState(t *testing.T) {
	r := newRig(t, jevtest.Status(500), allFields)
	verdict := atomicJudge(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.NotEmpty(t, verdict.State)
}

func atomicGateRig(t *testing.T, scores map[string]float64, mutate func(*config.Provider)) rig {
	t.Helper()
	return gateRig(t, atomicAnswer(t, scores), func(p *config.Provider) {
		p.Profile = calibration.ProfileAtomicV1
		if mutate != nil {
			mutate(p)
		}
	})
}

func atomicRecord(t *testing.T, threshold float64) calibration.Record {
	t.Helper()
	return pinnedRecord(t, threshold, func(m *calibration.Meta) {
		m.Profile, m.ProfileVersion = calibration.ProfileAtomicV1, ProfileVersionOf(calibration.ProfileAtomicV1, calibration.BoundaryPackage)
	})
}

func TestAnAtomicPrecheckApprovesWhenTheLowestCriterionClearsTheThreshold(t *testing.T) {
	r := atomicGateRig(t, allCriteria(0.9), nil)

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusApproved, report.Status)
	require.NotNil(t, report.Delegation)
	require.NoError(t, report.Delegation.Validate())
	assert.InDelta(t, 0.9, report.Confidence, 1e-9)
	assert.Equal(t, AggregateKey, report.Delegation.Criterion)
	assert.NotContains(t, r.sim.LastBody(), AggregateKey, "the provider is asked only the criterion questions")
}

func TestAnAtomicPrecheckSignalsTheCriterionThatFailsAndNeverApproves(t *testing.T) {
	scores := allCriteria(0.95)
	scores["validation_stated"] = 0.2
	r := atomicGateRig(t, scores, nil)

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusSignal, report.Status)
	require.Len(t, report.Hints, 1)
	assert.Equal(t, "validation_stated", report.Hints[0].Criterion)
	assert.InDelta(t, 0.2, report.Confidence, 1e-9)
	assert.Nil(t, report.Delegation)
}

func TestAnAtomicAnswerMissingACriterionNeverApproves(t *testing.T) {
	scores := allCriteria(0.99)
	delete(scores, "scope_bounded")
	r := atomicGateRig(t, scores, nil)
	report := r.consumer.Precheck(context.Background(), gateRequest())
	assert.Equal(t, StatusFallback, report.Status, "the adapter refuses an answer that skips a question it asked")
	assert.Nil(t, report.Delegation)
}

func TestAnAtomicPrecheckUnderAPinnedAtomicCalibrationUsesItsThreshold(t *testing.T) {
	record := atomicRecord(t, 0.62)
	r := atomicGateRig(t, allCriteria(0.7), func(p *config.Provider) { p.CalibrationID = record.ID })
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return record, nil }

	report := r.consumer.Precheck(context.Background(), gateRequest())

	require.Equal(t, StatusApproved, report.Status, "0.70 is above the calibrated 0.62 though below the default 0.80")
	assert.True(t, report.Calibration.Applied)
	assert.Equal(t, record.ID, report.Delegation.CalibrationID)
}

func TestACalibrationOfAnotherProfileDoesNotApplyToTheAtomicProfile(t *testing.T) {
	record := pinnedRecord(t, 0.62, nil) // a v1 record
	r := atomicGateRig(t, allCriteria(0.7), func(p *config.Provider) { p.CalibrationID = record.ID })
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return record, nil }

	report := r.consumer.Precheck(context.Background(), gateRequest())

	assert.False(t, report.Calibration.Applied)
	assert.Equal(t, StatusSignal, report.Status, "the default 0.80 stands")
}

func TestWithAggregateLeavesAPartialAtomicAnswerAndOtherProfilesAlone(t *testing.T) {
	partial := integration.Result{Answers: map[string]integration.Answer{"scope_bounded": {Kind: integration.KindNoul, Noul: 0.9}}}
	assert.Equal(t, partial, withAggregate(calibration.ProfileAtomicV1, TransitionArchivistToGate, partial))

	full := integration.Result{Answers: map[string]integration.Answer{}}
	for _, c := range Criteria(TransitionArchivistToGate) {
		full.Answers[c.Key] = integration.Answer{Kind: integration.KindNoul, Noul: 0.8}
	}
	assert.Equal(t, full, withAggregate(calibration.ProfileV2, TransitionArchivistToGate, full), "an aggregate-choice profile is not touched")
	combined := withAggregate(calibration.ProfileAtomicV1, TransitionArchivistToGate, full)
	assert.InDelta(t, 0.8, combined.Answers[AggregateKey].Probabilities[OptionAccept], 1e-9)
	assert.NotContains(t, full.Answers, AggregateKey, "the original answers are not mutated")
}
