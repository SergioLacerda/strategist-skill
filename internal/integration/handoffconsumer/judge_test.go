package handoffconsumer

import (
	"context"
	"net/http"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allFields(provider *config.Provider) {
	provider.DataPolicy.AllowedFields = append(Fields(handoff.TransitionRangerToArchivist), Fields(TransitionArchivistToGate)...)
}

func judgeOf(r rig) Judge {
	return Judge{Mechanism: r.consumer.Mechanism, Provider: r.consumer.Provider, Name: "jev", Version: "1"}
}

var packageProjection = calibration.Fields{"approved_scope": "- allowed: docs/**", "implementation_plan": "- [ ] 1.1 [analysis_artifact] Do it; verify it."}

func TestJudgeReturnsTheProvidersOwnNumbers(t *testing.T) {
	r := newRig(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(answerJSON(t, TransitionArchivistToGate, "jev-1.13.0", OptionAccept, 0.86))
	}, allFields)

	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)

	assert.Empty(t, verdict.State)
	assert.Equal(t, OptionAccept, verdict.Decision)
	assert.InDelta(t, 0.86, verdict.PAccept, 1e-9)
	assert.Equal(t, "jev-1.13.0", verdict.Model)
	assert.InDelta(t, 0.14, verdict.Probabilities[OptionReject], 1e-9)
}

func TestJudgeReportsARejectAsALowProbabilityOfAccept(t *testing.T) {
	r := newRig(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(answerJSON(t, TransitionArchivistToGate, "jev-1.13.0", OptionReject, 0.02))
	}, allFields)
	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, OptionReject, verdict.Decision)
	assert.InDelta(t, 0.02, verdict.PAccept, 1e-9)
}

func TestJudgeMapsProviderFailuresToTypedStates(t *testing.T) {
	r := newRig(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, allFields)
	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.NotEmpty(t, verdict.State)
	assert.Empty(t, verdict.Decision)
}

func TestJudgeNeverSendsAFieldTheDataPolicyDoesNotAllow(t *testing.T) {
	called := false
	r := newRig(t, func(_ http.ResponseWriter, _ *http.Request) { called = true }, func(p *config.Provider) { p.DataPolicy.AllowedFields = []string{"objective"} })
	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, "data_policy_denied", verdict.State)
	assert.False(t, called)
}

func TestJudgeOfADisabledProviderMakesNoCall(t *testing.T) {
	called := false
	r := newRig(t, func(_ http.ResponseWriter, _ *http.Request) { called = true }, func(p *config.Provider) { allFields(p); p.Enabled = false })
	verdict := judgeOf(r).Judge(context.Background(), calibration.BoundaryPackage, packageProjection)
	assert.Equal(t, "disabled", verdict.State)
	assert.False(t, called)
}

func TestProfileVersionIsStablePerBoundaryAndDiffersBetweenThem(t *testing.T) {
	first := ProfileVersion(calibration.BoundaryPackage)
	assert.Equal(t, first, ProfileVersion(calibration.BoundaryPackage))
	assert.NotEqual(t, ProfileVersion(calibration.BoundaryPackage), ProfileVersion(calibration.BoundaryRanger))
	assert.Len(t, ProfileVersion(calibration.BoundaryPackage), 8)
	require.Equal(t, TransitionArchivistToGate, TransitionForBoundary(calibration.BoundaryPackage))
	assert.Equal(t, handoff.TransitionRangerToArchivist, TransitionForBoundary(calibration.BoundaryRanger))
	b := Binding(calibration.BoundaryPackage, "m")
	assert.Equal(t, ProfileID, b.Profile)
	assert.Equal(t, "m", b.Model)
}
