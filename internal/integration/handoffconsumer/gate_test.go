package handoffconsumer

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/stretchr/testify/require"
)

func gateRequest() Request {
	return Request{MissionID: "m-1", Transition: TransitionArchivistToGate, Subject: "sha256:package", Source: gatePackageSource()}
}

func gatePackageSource() MapSource {
	return MapSource{
		"approved_scope":      "allowed: docs/**",
		"implementation_plan": "- [ ] 1.1 [analysis_artifact] note",
		"acceptance_checks":   "- tests pass",
	}
}

func gateAnswers(t *testing.T, choice string, confidence float64) []byte {
	t.Helper()
	return answerJSON(t, TransitionArchivistToGate, "jev-1.13.0", choice, confidence)
}

func gateRig(t *testing.T, body []byte, mutate func(*config.Provider)) rig {
	t.Helper()
	return newRig(t, jevtest.Reply(body), func(p *config.Provider) {
		p.DataPolicy.AllowedFields = Fields(TransitionArchivistToGate)
		if mutate != nil {
			mutate(p)
		}
	})
}

func TestTheGateBoundaryUsesTheRefinedPackageCatalogButIsNotALifecycleTransition(t *testing.T) {
	require.Equal(t, Criteria(handoff.TransitionArchivistToSniper), Criteria(TransitionArchivistToGate), "the same package, the same criteria")
	require.Equal(t, Fields(handoff.TransitionArchivistToSniper), Fields(TransitionArchivistToGate))
	require.NotEqual(t, handoff.TransitionArchivistToSniper, TransitionArchivistToGate)
}

func TestAnApprovedGateCheckIsRecordedWithItsProvenanceAndReusedForTheSameRevision(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), nil)

	first := r.consumer.Precheck(context.Background(), gateRequest())
	require.Equal(t, StatusApproved, first.Status)
	require.False(t, first.Reused)
	require.Equal(t, 1, r.sim.Calls())

	later := gateRequest()
	later.Transition = handoff.TransitionArchivistToSniper
	reused, ok := r.consumer.Reuse(later)
	require.True(t, ok, "the later evaluation of the same revision reuses the pre-gate reading")
	require.Equal(t, StatusApproved, reused.Status)
	require.True(t, reused.Reused)
	require.NotNil(t, reused.Delegation)
	require.NoError(t, reused.Delegation.Validate())
	require.Equal(t, "sha256:package", reused.Delegation.Subject)
	require.Equal(t, first.Delegation.BindingDigest, reused.Delegation.BindingDigest)
	require.InDelta(t, 0.96, reused.Delegation.Confidence, 1e-9)
	require.Equal(t, 1, r.sim.Calls(), "reuse makes no second provider call")
}

func TestAChangedRevisionIsNeverReused(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), nil)
	r.consumer.Precheck(context.Background(), gateRequest())

	changed := gateRequest()
	changed.Transition = handoff.TransitionArchivistToSniper
	changed.Subject = "sha256:edited-after-the-gate-check"
	_, ok := r.consumer.Reuse(changed)
	require.False(t, ok, "a package that changed is asked again")
}

func TestOnlyTheSniperEvaluationReusesAndOnlyAnAnsweredCheck(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), nil)
	r.consumer.Precheck(context.Background(), gateRequest())

	ranger := gateRequest()
	ranger.Transition = handoff.TransitionRangerToArchivist
	_, ok := r.consumer.Reuse(ranger)
	require.False(t, ok, "no other transition borrows the package reading")

	failing := gateRig(t, nil, nil)
	failing.consumer.Precheck(context.Background(), gateRequest())
	later := gateRequest()
	later.Transition = handoff.TransitionArchivistToSniper
	_, ok = failing.consumer.Reuse(later)
	require.False(t, ok, "a fallback has no answer to reuse")
}

func TestASignalIsReusedAsASignalWithItsHints(t *testing.T) {
	body := answerJSON(t, TransitionArchivistToGate, "jev-1.13.0", OptionAccept, 0.55, "validation_stated")
	r := gateRig(t, body, nil)
	first := r.consumer.Precheck(context.Background(), gateRequest())
	require.Equal(t, StatusSignal, first.Status)

	later := gateRequest()
	later.Transition = handoff.TransitionArchivistToSniper
	reused, ok := r.consumer.Reuse(later)
	require.True(t, ok)
	require.Equal(t, StatusSignal, reused.Status)
	require.Nil(t, reused.Delegation)
	require.Equal(t, []string{"validation_stated"}, hintKeys(reused.Hints))
}

func TestATamperedLedgerRecordIsNeverReused(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), nil)
	r.consumer.Precheck(context.Background(), gateRequest())
	raw := readFile(t, r.ledger)
	writeFile(t, r.ledger, replaceOnce(raw, `"confidence":0.96`, `"confidence":0.99`))

	later := gateRequest()
	later.Transition = handoff.TransitionArchivistToSniper
	_, ok := r.consumer.Reuse(later)
	require.False(t, ok, "an edited record fails its integrity digest")
}

func TestReuseIsRefusedWhenTheIntegrationIsDisabledNow(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), nil)
	r.consumer.Precheck(context.Background(), gateRequest())
	r.consumer.Provider.Enabled = false

	later := gateRequest()
	later.Transition = handoff.TransitionArchivistToSniper
	_, ok := r.consumer.Reuse(later)
	require.False(t, ok)
}

func TestTheGateCheckSpendsItsOwnBudgetPerRevisionAndTransition(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.96), func(p *config.Provider) { p.CallBudget = 1 })
	require.Equal(t, StatusApproved, r.consumer.Precheck(context.Background(), gateRequest()).Status)
	second := r.consumer.Precheck(context.Background(), gateRequest())
	require.Equal(t, StatusFallback, second.Status, "the budget for archivist_to_gate is spent")

	sniper := gateRequest()
	sniper.Transition = handoff.TransitionArchivistToSniper
	_, ok := r.consumer.Reuse(sniper)
	require.True(t, ok, "reuse does not need budget")
}
