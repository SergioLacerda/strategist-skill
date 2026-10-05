package handoffconsumer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var partial = MapSource{"implementation_plan": "- [ ] 1.1 [analysis_artifact] note; verify it."}

func TestProfileV1ProjectsOnlyThePresentFieldsAsBefore(t *testing.T) {
	text, err := Project(TransitionArchivistToGate, partial, Fields(TransitionArchivistToGate), 4096)
	require.NoError(t, err)
	assert.NotContains(t, text, "(absent)")
	assert.NotContains(t, text, "approved_scope")
	same, err := ProjectFor("", TransitionArchivistToGate, partial, Fields(TransitionArchivistToGate), 4096)
	require.NoError(t, err)
	assert.Equal(t, text, same, "the empty profile is v1")
}

func TestProfileV2ShowsAMissingAllowedFieldAsAbsentWithoutAnyContent(t *testing.T) {
	text, err := ProjectFor(calibration.ProfileV2, TransitionArchivistToGate, partial, Fields(TransitionArchivistToGate), 4096)
	require.NoError(t, err)
	assert.Contains(t, text, "## approved_scope\n(absent)")
	assert.Contains(t, text, "## acceptance_checks\n(absent)")
	assert.Contains(t, text, "## implementation_plan\n- [ ] 1.1")

	narrow, err := ProjectFor(calibration.ProfileV2, TransitionArchivistToGate, partial, []string{"implementation_plan"}, 4096)
	require.NoError(t, err)
	assert.NotContains(t, narrow, "approved_scope", "a field the data policy does not allow is never mentioned, not even as absent")
}

func TestProfileV2StillDeniesAnArtifactWithNothingToSend(t *testing.T) {
	_, err := ProjectFor(calibration.ProfileV2, TransitionArchivistToGate, MapSource{}, Fields(TransitionArchivistToGate), 4096)
	require.Error(t, err)
	_, err = ProjectFor(calibration.ProfileV2, TransitionArchivistToGate, partial, Fields(TransitionArchivistToGate), 10)
	require.Error(t, err, "an oversized projection is denied, not truncated")
}

func TestProfileV2ExplainsWhatAbsentMeansAndHasItsOwnVersion(t *testing.T) {
	v1 := QuestionsFor(calibration.ProfileV1, TransitionArchivistToGate)
	v2 := QuestionsFor(calibration.ProfileV2, TransitionArchivistToGate)
	assert.NotContains(t, v1[0].Instructions, "(absent)")
	assert.Contains(t, v2[0].Instructions, "A field shown as (absent) was not provided")
	assert.Len(t, v2, len(v1))

	for _, boundary := range []string{calibration.BoundaryPackage, calibration.BoundaryRanger} {
		assert.NotEqual(t, ProfileVersionOf(calibration.ProfileV1, boundary), ProfileVersionOf(calibration.ProfileV2, boundary))
		assert.Equal(t, ProfileVersion(boundary), ProfileVersionOf("", boundary), "the default is v1")
	}
}

func TestTheV1VersionIsTheDigestOfTheQuestionsAloneSoExistingCalibrationsStayValid(t *testing.T) {
	raw, err := json.Marshal(Questions(TransitionArchivistToGate))
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(raw))[:8], ProfileVersion(calibration.BoundaryPackage))
}

func TestABindingCarriesTheProfileAndItsVersion(t *testing.T) {
	b := BindingFor("", calibration.BoundaryPackage, "m")
	assert.Equal(t, calibration.ProfileV1, b.Profile)
	b = BindingFor(calibration.ProfileV2, calibration.BoundaryPackage, "m")
	assert.Equal(t, calibration.ProfileV2, b.Profile)
	assert.Equal(t, ProfileVersionOf(calibration.ProfileV2, calibration.BoundaryPackage), b.ProfileVersion)
}

func TestAProviderOnProfileV2SendsAbsentMarkersAndNeedsAV2Calibration(t *testing.T) {
	record := pinnedRecord(t, 0.60, nil) // a v1 calibration
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.70), func(p *config.Provider) { p.Profile = calibration.ProfileV2; p.CalibrationID = record.ID })
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return record, nil }
	source := MapSource{"implementation_plan": "- [ ] 1.1 [analysis_artifact] note; verify it."}

	report := r.consumer.Precheck(context.Background(), Request{MissionID: "m-1", Transition: TransitionArchivistToGate, Subject: "sha256:p", Source: source})

	assert.Contains(t, r.sim.LastBody(), "(absent)")
	assert.False(t, report.Calibration.Applied, "a v1 calibration never serves a v2 provider")
	assert.Equal(t, string(calibration.ReasonProfileChanged), report.Calibration.Reason)
	assert.Equal(t, StatusSignal, report.Status, "0.70 under the default 0.80")

	v2 := pinnedRecord(t, 0.60, func(m *calibration.Meta) {
		m.Profile = calibration.ProfileV2
		m.ProfileVersion = ProfileVersionOf(calibration.ProfileV2, calibration.BoundaryPackage)
		m.Seed = 9
	})
	r.consumer.Provider.CalibrationID = v2.ID
	r.consumer.Calibrations = func(string) (calibration.Record, error) { return v2, nil }
	assert.Equal(t, StatusApproved, r.consumer.Precheck(context.Background(), Request{MissionID: "m-2", Transition: TransitionArchivistToGate, Subject: "sha256:p", Source: source}).Status)
}

func TestTheJudgeAsksTheProfileItWasGivenWhateverTheProviderIsConfiguredFor(t *testing.T) {
	r := gateRig(t, gateAnswers(t, OptionAccept, 0.80), nil)
	j := judgeOf(r)
	j.Profile = calibration.ProfileV2
	verdict := j.Judge(context.Background(), calibration.BoundaryPackage, calibration.Fields{"implementation_plan": "- [ ] 1.1 [analysis_artifact] x; verify y."})
	assert.Empty(t, verdict.State)
	assert.Contains(t, r.sim.LastBody(), "(absent)")

	j.Profile = ""
	_ = j.Judge(context.Background(), calibration.BoundaryPackage, calibration.Fields{"implementation_plan": "- [ ] 1.1 [analysis_artifact] x; verify y."})
	assert.NotContains(t, r.sim.LastBody(), "(absent)")
}

func TestTheConfigAcceptsOnlyAKnownProfile(t *testing.T) {
	for profile, ok := range map[string]bool{"": true, "handoff-v1": true, "handoff-v2": true, "handoff-v3": false, "v2": false} {
		file := config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": {Profile: profile}}}
		if ok {
			require.NoError(t, file.Validate(), profile)
		} else {
			require.Error(t, file.Validate(), profile)
		}
	}
}
