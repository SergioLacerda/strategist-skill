package main

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalibrationDependenciesAreComplete(t *testing.T) {
	deps := calibrationDependencies()
	require.NotNil(t, deps.Reader)
	require.NotNil(t, deps.Judge)
	require.NotNil(t, deps.Current)
	require.NotNil(t, deps.Plan)
	require.NotNil(t, deps.Signals)
	require.NotNil(t, deps.Policy)
	require.NotNil(t, deps.Endpoint)
	require.NotNil(t, deps.LivePlan)
	require.NotNil(t, deps.Live)
	require.NotNil(t, deps.PairJudge)
	require.NotNil(t, deps.PairProfile)
	pairID, pairVersion := deps.PairProfile()
	assert.Equal(t, handoffconsumer.PairProfileID, pairID)
	assert.Equal(t, handoffconsumer.PairProfileVersion(), pairVersion)
	id, version := deps.Profile("", calibration.BoundaryPackage)
	assert.Equal(t, handoffconsumer.ProfileID, id)
	assert.Equal(t, handoffconsumer.ProfileVersion(calibration.BoundaryPackage), version)
}

func TestTheCalibrationJudgeNeedsAnEnabledIntegration(t *testing.T) {
	root := bareRoot(t)
	_, err := calibrationJudge(root, "")
	require.ErrorContains(t, err, "no provider integration is enabled")
}

func TestTheCalibrationJudgeAsksTheProviderThroughTheMechanism(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.86)))
	enableIntegration(t, root, sim, func(p *config.Provider) {
		p.DataPolicy.AllowedFields = handoffconsumer.Fields(handoffconsumer.TransitionArchivistToGate)
	})
	judge, err := calibrationJudge(root, "")
	require.NoError(t, err)

	verdict := judge.Judge(context.Background(), calibration.BoundaryPackage, calibration.Fields{"approved_scope": "allowed: docs/**", "implementation_plan": "- [ ] 1.1 [analysis_artifact] x; verify y."})

	require.Empty(t, verdict.State)
	assert.InDelta(t, 0.86, verdict.PAccept, 1e-9)
	assert.Equal(t, "jev-1.13.0", verdict.Model)
	assert.Equal(t, 1, sim.Calls())
}

func TestTheConfiguredThresholdFollowsTheIntegrationFile(t *testing.T) {
	root := bareRoot(t)
	assert.InDelta(t, config.DefaultThreshold, configuredThreshold(root), 1e-9)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.9)))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.ApprovalThreshold = 0.91 })
	assert.InDelta(t, 0.91, configuredThreshold(root), 1e-9)
}

func TestThePlanUsesTheDataPolicyOfTheEnabledProvider(t *testing.T) {
	cases := []calibration.Case{{ID: "a", Fields: calibration.Fields{"approved_scope": "allowed: docs/**", "implementation_plan": "plan text", "acceptance_checks": "- ok"}}, {ID: "b", Fields: calibration.Fields{"unrelated": "x"}}}

	root := bareRoot(t)
	plan, err := calibrationPlan(root, "", calibration.BoundaryPackage, cases)
	require.NoError(t, err)
	assert.Equal(t, []string{"approved_scope", "implementation_plan", "acceptance_checks"}, plan.Fields, "without an integration the whole catalog is shown")
	assert.Equal(t, 1, plan.Denied)
	assert.Positive(t, plan.Bytes)

	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.9)))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.DataPolicy.AllowedFields = []string{"implementation_plan"} })
	plan, err = calibrationPlan(root, "", calibration.BoundaryPackage, cases)
	require.NoError(t, err)
	assert.Equal(t, []string{"implementation_plan"}, plan.Fields)
}

func TestMissionSignalsReadTheLedgerWithoutContent(t *testing.T) {
	root := bareRoot(t)
	ledger := handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}
	require.NoError(t, ledger.Append(handoffconsumer.Record{MissionID: "m1", Transition: handoffconsumer.TransitionArchivistToGate, Subject: "sha256:p", Provider: "jev", Attempted: true, Answered: true, Approved: false, Confidence: 0.4, Threshold: 0.8}))
	require.NoError(t, ledger.Append(handoffconsumer.Record{MissionID: "m1", Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:a", Provider: "jev", Attempted: true, Answered: true, Approved: true, Confidence: 0.9, Threshold: 0.8}))

	signals, err := missionSignals(root, "m1")

	require.NoError(t, err)
	require.Len(t, signals, 2)
	assert.False(t, signals[1].Approved)
	assert.True(t, signals[0].Approved)
	assert.Empty(t, signals[0].GateOutcome, "no gate outcome is recorded for this mission yet")
	none, err := missionSignals(root, "other")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestThePolicyItemsAllPass(t *testing.T) {
	items := policyItems(timeNow())
	require.NotEmpty(t, items)
	for _, item := range items {
		assert.True(t, item.Passed, "%s: %s", item.Label, item.Detail)
		assert.Contains(t, item.Label, "calibration policy: ")
	}
}

func timeNow() time.Time { return time.Now() }

func TestTheEndpointAndTheLivePlanFeedTheConsentTerm(t *testing.T) {
	root := bareRoot(t)
	assert.Equal(t, "https://api.typesafe.ai/v1/systemone", calibrationEndpoint(root), "the documented endpoint when nothing is configured")
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.9)))
	enableIntegration(t, root, sim, nil)
	assert.Contains(t, calibrationEndpoint(root), sim.URL())
	calls, bytes := livePlan()
	assert.Equal(t, 3, calls)
	assert.Positive(t, bytes)
}

func TestTheLiveSmokeRunsThroughTheSameMechanismAndNeverWithoutAnEnabledIntegration(t *testing.T) {
	root := bareRoot(t)
	_, err := runLiveSmoke(context.Background(), root)
	require.ErrorContains(t, err, "no provider integration is enabled")

	// The smoke check calls in a fixed order: the probe, the well-formed handoff, the vague one.
	var calls int
	sim := jevtest.Start(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		switch calls {
		case 1:
			_, _ = w.Write([]byte(probeBody))
		case 2:
			_, _ = w.Write(answerFor(t, handoff.TransitionRangerToArchivist, "accept", 0.9))
		default:
			_, _ = w.Write(answerFor(t, handoff.TransitionRangerToArchivist, "reject", 0.1))
		}
	})
	enableIntegration(t, root, sim, func(p *config.Provider) {
		p.DataPolicy.AllowedFields = handoffconsumer.Fields(handoff.TransitionRangerToArchivist)
	})

	items, err := runLiveSmoke(context.Background(), root)

	require.NoError(t, err)
	require.Len(t, items, 7)
	for _, item := range items {
		assert.True(t, item.Passed, "%s: %s", item.Label, item.Detail)
	}
	assert.Equal(t, 3, sim.Calls())
}

func TestThePairJudgeNeedsAnEnabledIntegrationAndUsesTheMechanism(t *testing.T) {
	root := bareRoot(t)
	_, err := calibrationPairJudge(root)
	require.ErrorContains(t, err, "no provider integration is enabled")

	sim := jevtest.Start(t, jevtest.Reply([]byte(`{"model":"jev-1.13.0","answers":{"task_kind_consistent":{"type":"noul","noul":0.9},"obligation_preserved":{"type":"noul","noul":0.8},"verification_checks_task":{"type":"noul","noul":0.7}},"usage":{"input_tokens":150,"output_tokens":9}}`)))
	enableIntegration(t, root, sim, nil)
	judge, err := calibrationPairJudge(root)
	require.NoError(t, err)

	verdict := judge.JudgePair(context.Background(), calibration.SyntheticPairs(1)[0])

	require.Empty(t, verdict.State)
	assert.InDelta(t, 0.8, verdict.Scores[calibration.DimObligation], 1e-9)
	assert.Equal(t, 150, verdict.InputTokens)
	assert.Equal(t, 1, sim.Calls())
}
