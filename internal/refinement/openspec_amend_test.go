package refinement

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var amendClock = func() time.Time { return time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC) }

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// publishedAt publishes change "first" and sets the analysis status (and an optional
// extra frontmatter line), returning the fixture.
func publishedAt(t *testing.T, status, extra string) bridgeFixture {
	t.Helper()
	f := newBridgeFixture(t)
	f.change(t, "first", "v1")
	_, err := f.normalize("first")
	require.NoError(t, err)
	path := filepath.Join(f.refined, "analysis.md")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	text := strings.Replace(string(raw), "mission_status: archivist_done", "mission_status: "+status+extra, 1)
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
	return f
}

func (f bridgeFixture) amendInput(change, amends string) AmendInput {
	raw, _ := os.ReadFile(filepath.Join(f.refined, "analysis.md"))
	status := frontmatterValue(raw, "mission_status")
	persisted := domain.MissionEngineStatus{MissionID: f.mission, Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	if status == "gate_pending" {
		persisted.Phase, persisted.State = domain.PhaseApprovalGate, domain.StateApprovalGate
	}
	if status == "gate_analysis_accepted" {
		persisted.Phase, persisted.State = domain.PhaseDone, domain.StateDoneAnalysis
	}
	return AmendInput{MissionID: f.mission, BasePath: f.base, RuntimeRoot: f.runtime, ChangeID: change, Amends: amends, AuthorizationRef: "user: amend it (2026-09-26)", GateLabel: "", PersistedStatus: persisted, Now: amendClock}
}

// newChange writes the amending change (the pending analysis stays absent: an amend
// never consumes one).
func (f bridgeFixture) newChange(t *testing.T, id, body string) {
	t.Helper()
	f.change(t, id, body)
	require.NoError(t, os.Remove(f.pending))
}

func assertUntouched(t *testing.T, f bridgeFixture, before map[string]string, change string) {
	t.Helper()
	assert.Equal(t, before, readAll(t, f.refined), "the package bytes are unchanged")
	_, err := os.Stat(filepath.Join(f.refined, ".amendments"))
	require.ErrorIs(t, err, os.ErrNotExist, "no snapshot was written")
	_, err = os.Stat(filepath.Join(f.runtime, "changes", change))
	assert.NoError(t, err, "the change is not archived")
}

func TestAmendOpenSpecRefusesEveryUnmetPrecondition(t *testing.T) {
	type mutate func(t *testing.T, f bridgeFixture, in *AmendInput)
	cases := map[string]struct {
		status, extra string
		mutate        mutate
		want          string
	}{
		"status ranger_pending":        {status: "ranger_pending", want: "mission_status"},
		"status ranger_done":           {status: "ranger_done", want: "mission_status"},
		"status archivist_pending":     {status: "archivist_pending", want: "mission_status"},
		"status sniper_running":        {status: "sniper_running", want: "mission_status"},
		"status documentation_applied": {status: "documentation_applied", want: "mission_status"},
		"status gate_rejected":         {status: "gate_rejected", want: "mission_status"},
		"status unknown":               {status: "mystery", want: "mission_status"},
		"claimed_by present":           {status: "gate_analysis_accepted", extra: "\nclaimed_by: sniper", want: "claimed_by"},
		"amends mismatch":              {status: "archivist_done", mutate: func(_ *testing.T, _ bridgeFixture, in *AmendInput) { in.Amends = "other" }, want: "--amends"},
		"empty reference":              {status: "archivist_done", mutate: func(_ *testing.T, _ bridgeFixture, in *AmendInput) { in.AuthorizationRef = "" }, want: "authorization reference"},
		"multi-line reference":         {status: "archivist_done", mutate: func(_ *testing.T, _ bridgeFixture, in *AmendInput) { in.AuthorizationRef = "a\nb" }, want: "authorization reference"},
		"control character reference":  {status: "archivist_done", mutate: func(_ *testing.T, _ bridgeFixture, in *AmendInput) { in.AuthorizationRef = "a\x07b" }, want: "authorization reference"},
		"gate label rejected":          {status: "archivist_done", mutate: func(_ *testing.T, _ bridgeFixture, in *AmendInput) { in.GateLabel = "rejected" }, want: "rejected"},
		"pending analysis present": {status: "archivist_done", mutate: func(t *testing.T, f bridgeFixture, _ *AmendInput) {
			require.NoError(t, os.MkdirAll(filepath.Dir(f.pending), 0o755))
			require.NoError(t, os.WriteFile(f.pending, []byte("---\nmission_id: m-1\n---\n"), 0o644))
		}, want: "pending analysis"},
		"incomplete change": {status: "archivist_done", mutate: func(t *testing.T, f bridgeFixture, _ *AmendInput) {
			require.NoError(t, os.Remove(filepath.Join(f.runtime, "changes", "second", "tasks.md")))
		}, want: "incomplete change"},
		"wrong mission identity": {status: "archivist_done", mutate: func(t *testing.T, f bridgeFixture, _ *AmendInput) {
			path := filepath.Join(f.refined, "analysis.md")
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "mission_id: m-1", "mission_id: other", 1)), 0o644))
		}, want: "mission_id"},
		"accepted package gains documentation targets": {status: "gate_analysis_accepted", mutate: func(t *testing.T, f bridgeFixture, _ *AmendInput) {
			path := filepath.Join(f.runtime, "changes", "second", "tasks.md")
			require.NoError(t, os.WriteFile(path, []byte("- [ ] 1.1 [documentation_target] write it\n"), 0o644))
		}, want: "documentation_target"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := publishedAt(t, tc.status, tc.extra)
			f.newChange(t, "second", "v2")
			in := f.amendInput("second", "first")
			if tc.mutate != nil {
				tc.mutate(t, f, &in)
			}
			before := readAll(t, f.refined)

			_, err := AmendOpenSpec(in)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assertUntouched(t, f, before, "second")
		})
	}
}

func TestAmendOpenSpecRefusesAMissingPackage(t *testing.T) {
	f := newBridgeFixture(t)
	f.newChange(t, "second", "v2")

	_, err := AmendOpenSpec(f.amendInput("second", "first"))

	require.ErrorContains(t, err, "no refined package")
}

func TestAmendOpenSpecAcceptsEveryAllowedStatusAndKeepsTheAnalysisByteIdentical(t *testing.T) {
	for _, status := range []string{"archivist_done", "gate_pending", "gate_revision_requested", "gate_analysis_accepted"} {
		t.Run(status, func(t *testing.T) {
			f := publishedAt(t, status, "")
			f.newChange(t, "second", "v2")
			analysis := readAll(t, f.refined)["analysis.md"]

			result, err := AmendOpenSpec(f.amendInput("second", "first"))

			require.NoError(t, err)
			assert.Equal(t, 1, result.Amendment)
			assert.Equal(t, status, result.Status, "the status is unchanged")
			after := readAll(t, f.refined)
			assert.Equal(t, sha([]byte(analysis)), sha([]byte(after["analysis.md"])), "analysis.md is byte-identical")
			assert.Contains(t, after["analysis.md"], "provider_change_id: first", "the original provider_change_id survives")
			for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
				assert.Contains(t, after[name], "v2", "%s is replaced", name)
			}
		})
	}
}

func TestAmendOpenSpecRejectsBlockedStateWithoutRepair(t *testing.T) {
	f := publishedAt(t, "gate_analysis_accepted", "")
	f.newChange(t, "second", "v2")
	in := f.amendInput("second", "first")
	in.PersistedStatus.Phase, in.PersistedStatus.State = domain.PhaseBlocked, domain.StateBlocked
	_, err := AmendOpenSpec(in)
	require.ErrorContains(t, err, "disagrees")
}

func TestAmendOpenSpecRejectsDivergentGateTelemetry(t *testing.T) {
	f := publishedAt(t, "archivist_done", "")
	f.newChange(t, "second", "v2")
	in := f.amendInput("second", "first")
	in.GateLabel = "revision_requested"
	_, err := AmendOpenSpec(in)
	require.ErrorContains(t, err, "telemetry label")
}

func TestAmendOpenSpecAllowsARepairAuthorizedState(t *testing.T) {
	f := publishedAt(t, "gate_analysis_accepted", "")
	f.newChange(t, "second", "v2")
	require.NoError(t, os.WriteFile(filepath.Join(f.refined, "tasks.md"), []byte("- [ ] 1.1 [documentation_target] write `docs/repaired.md`\n"), 0o644))
	in := f.amendInput("second", "first")
	in.PersistedStatus.Phase, in.PersistedStatus.State = domain.PhaseRefinement, domain.StateRefinement
	in.PersistedStatus.HandoffNextAction = "reapprove_gate"
	_, err := AmendOpenSpec(in)
	require.NoError(t, err)
}

func TestAmendOpenSpecRecordsAmendmentsInFrontmatterAndManifest(t *testing.T) {
	f := publishedAt(t, "gate_analysis_accepted", "")
	f.newChange(t, "second", "v2")
	previous := readAll(t, f.refined)

	result, err := AmendOpenSpec(f.amendInput("second", "first"))
	require.NoError(t, err)

	tasks := readAll(t, f.refined)["tasks.md"]
	assert.True(t, strings.HasPrefix(tasks, "---\namendments:\n"), "an amendments frontmatter is added: %q", tasks)
	assert.Contains(t, tasks, "amendment: 1")
	assert.Contains(t, tasks, "change_id: second")
	assert.Contains(t, tasks, "at: 2026-09-26T10:00:00Z")
	assert.Contains(t, tasks, `authorization_ref: "user: amend it (2026-09-26)"`)
	assert.Contains(t, tasks, "derived_from: first")
	assert.Contains(t, tasks, "source_digest:")
	assert.Contains(t, tasks, "package_digest: sha256:")
	assert.Contains(t, tasks, "reason: same_mission_amendment")
	assert.Contains(t, tasks, "disposition: same_mission_amendment")
	assert.Contains(t, tasks, "previous_sha256: "+sha([]byte(previous["tasks.md"])))

	snapshot := filepath.Join(f.refined, ".amendments", "001")
	assert.Equal(t, snapshot, result.AmendmentDir)
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		raw, readErr := os.ReadFile(filepath.Join(snapshot, name))
		require.NoError(t, readErr)
		assert.Equal(t, previous[name], string(raw), "the previous %s is snapshotted", name)
	}
	var manifest amendmentManifest
	raw, err := os.ReadFile(filepath.Join(snapshot, "manifest.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &manifest))
	assert.Equal(t, 1, manifest.Amendment)
	assert.Equal(t, "second", manifest.ChangeID)
	assert.Equal(t, "first", manifest.Amends)
	assert.Equal(t, "first", manifest.DerivedFrom)
	assert.Empty(t, manifest.SupersedesMission)
	assert.NotEmpty(t, manifest.SourceDigest)
	assert.Contains(t, manifest.PackageDigest, "sha256:")
	assert.Equal(t, "same_mission_amendment", manifest.Reason)
	assert.Equal(t, "same_mission_amendment", manifest.Disposition)
	assert.Equal(t, "none", manifest.GateLabel)
	assert.Equal(t, "user: amend it (2026-09-26)", manifest.AuthorizationRef)
	assert.Equal(t, sha([]byte(previous["analysis.md"])), manifest.AnalysisSHA256)
	assert.Equal(t, sha([]byte(previous["tasks.md"])), manifest.Files["tasks.md"].PreviousSHA256)
	assert.Equal(t, sha([]byte(tasks)), manifest.Files["tasks.md"].NewSHA256)
}

func TestAmendOpenSpecArchivesTheNewChangeAndChainsTheNextAmendment(t *testing.T) {
	f := publishedAt(t, "archivist_done", "")
	f.newChange(t, "second", "v2")
	_, err := AmendOpenSpec(f.amendInput("second", "first"))
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(f.runtime, "changes", "second"))
	require.ErrorIs(t, statErr, os.ErrNotExist, "the amending change leaves the active list")
	archived, err := filepath.Glob(filepath.Join(f.runtime, "changes", "archive", "*-second"))
	require.NoError(t, err)
	assert.Len(t, archived, 1)

	f.newChange(t, "third", "v3")
	_, err = AmendOpenSpec(f.amendInput("third", "first"))
	require.ErrorContains(t, err, "--amends", "the second amendment must name the first amendment's change")

	result, err := AmendOpenSpec(f.amendInput("third", "second"))
	require.NoError(t, err)
	assert.Equal(t, 2, result.Amendment)
	tasks := readAll(t, f.refined)["tasks.md"]
	assert.Contains(t, tasks, "amendment: 1", "the earlier amendment entry is kept (append-only)")
	assert.Contains(t, tasks, "amendment: 2")
	_, err = os.Stat(filepath.Join(f.refined, ".amendments", "002", "manifest.yaml"))
	assert.NoError(t, err)
}

func TestAmendOpenSpecRestoresThePackageWhenAReplacementFails(t *testing.T) {
	f := publishedAt(t, "archivist_done", "")
	f.newChange(t, "second", "v2")
	before := readAll(t, f.refined)
	in := f.amendInput("second", "first")
	replaced := 0
	in.replaceHook = func(string) error {
		replaced++
		if replaced == 2 {
			return errors.New("injected failure")
		}
		return nil
	}

	_, err := AmendOpenSpec(in)

	require.ErrorContains(t, err, "injected failure")
	assert.Equal(t, before, readAll(t, f.refined), "the three files are restored from the snapshot")
	_, statErr := os.Stat(filepath.Join(f.runtime, "changes", "second"))
	assert.NoError(t, statErr, "the change is not archived after a failed amendment")
}
