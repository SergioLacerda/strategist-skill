package mission_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

// normalizeFixture is a project with one complete OpenSpec change and its pending
// analysis, wired through injected path resolution.
func normalizeFixture(t *testing.T) (mission.NormalizeDependencies, string) {
	t.Helper()
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	change := filepath.Join(runtime, "changes", "c-1")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(change, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\nmission_status: archivist_pending\n---\n\n# Analysis\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(change, name), []byte("# "+name+"\n"), 0o644))
	}
	deps := mission.NormalizeDependencies{
		RootFlag:         "root",
		RequireMissionID: func(string) error { return nil },
		ResolvePaths: func(mission.NormalizeOptions) (string, string, string, error) {
			return base, runtime, pending, nil
		},
		RecordConfidence: func(mission.NormalizeOptions, domain.ConfidenceClaim, []domain.Evidence) error { return nil },
	}
	return deps, filepath.Join(base, "refined", "m-1")
}

func runNormalize(t *testing.T, deps mission.NormalizeDependencies, args ...string) (string, error) {
	t.Helper()
	cmd := mission.NewNormalizeOpenSpec(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// M016 pin: the default invocation prints exactly this line.
func TestNormalizeOpenSpecDefaultOutputLineIsPinned(t *testing.T) {
	deps, refined := normalizeFixture(t)
	var recorded domain.ConfidenceClaim
	deps.RecordConfidence = func(_ mission.NormalizeOptions, claim domain.ConfidenceClaim, _ []domain.Evidence) error {
		recorded = claim
		return nil
	}

	out, err := runNormalize(t, deps, "--mission-id", "m-1", "--change-id", "c-1")

	require.NoError(t, err)
	require.Equal(t, "mission_id=m-1 provider_change_id=c-1 refined="+refined+" status=archivist_done\n", out)
	require.Equal(t, "archivist", recorded.Agent)
}

// amendFixture publishes change c-1 through the default path, moves the package to
// gate_analysis_accepted, and prepares the amending change c-2 (no pending analysis:
// an amend never consumes one).
func amendFixture(t *testing.T) (mission.NormalizeDependencies, string) {
	t.Helper()
	deps, refined := normalizeFixture(t)
	_, err := runNormalize(t, deps, "--mission-id", "m-1", "--change-id", "c-1")
	require.NoError(t, err)
	analysis := filepath.Join(refined, "analysis.md")
	raw, err := os.ReadFile(analysis)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(analysis, []byte(strings.Replace(string(raw), "mission_status: archivist_done", "mission_status: gate_analysis_accepted", 1)), 0o644))
	_, runtime, _, err := deps.ResolvePaths(mission.NormalizeOptions{})
	require.NoError(t, err)
	change := filepath.Join(runtime, "changes", "c-2")
	require.NoError(t, os.MkdirAll(change, 0o755))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(change, name), []byte("# "+name+" v2\n"), 0o644))
	}
	return deps, refined
}

func TestNormalizeOpenSpecAmendAppliesAndReportsTheAmendment(t *testing.T) {
	deps, refined := amendFixture(t)

	out, err := runNormalize(t, deps, "--mission-id", "m-1", "--change-id", "c-2", "--amend", "--amends", "c-1", "--authorization-ref", "user, 2026-09-26")

	require.NoError(t, err)
	require.Equal(t, "mission_id=m-1 provider_change_id=c-1 refined="+refined+" amendment=001 status=gate_analysis_accepted\n", out)
	tasks, err := os.ReadFile(filepath.Join(refined, "tasks.md"))
	require.NoError(t, err)
	require.Contains(t, string(tasks), "v2")
	_, err = os.Stat(filepath.Join(refined, ".amendments", "001", "manifest.yaml"))
	require.NoError(t, err)
}

func TestNormalizeOpenSpecAmendFlagsAreValidated(t *testing.T) {
	deps, _ := amendFixture(t)
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"amends needs amend":    {[]string{"--amends", "c-1"}, "--amends requires --amend"},
		"reference needs amend": {[]string{"--authorization-ref", "x"}, "--authorization-ref requires --amend"},
		"pending is refused":    {[]string{"--amend", "--amends", "c-1", "--authorization-ref", "x", "--pending-analysis", "p.md"}, "--pending-analysis cannot be used with --amend"},
		"amends is required":    {[]string{"--amend", "--authorization-ref", "x"}, "--amends"},
		"reference is required": {[]string{"--amend", "--amends", "c-1"}, "--authorization-ref"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := runNormalize(t, deps, append([]string{"--mission-id", "m-1", "--change-id", "c-2"}, tc.args...)...)

			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestNormalizeOpenSpecAmendRefusesARejectedGateLabel(t *testing.T) {
	deps, _ := amendFixture(t)
	deps.GateLabel = func(mission.NormalizeOptions) (string, error) { return "rejected", nil }

	_, err := runNormalize(t, deps, "--mission-id", "m-1", "--change-id", "c-2", "--amend", "--amends", "c-1", "--authorization-ref", "x")

	require.ErrorContains(t, err, "rejected")
}

// The default invocation without the new flags still fails closed on a differing
// package, exactly as before.
func TestNormalizeOpenSpecDefaultStillFailsClosedOnADifferingPackage(t *testing.T) {
	deps, _ := amendFixture(t)
	_, _, pending, err := deps.ResolvePaths(mission.NormalizeOptions{})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\n---\n"), 0o644))

	_, err = runNormalize(t, deps, "--mission-id", "m-1", "--change-id", "c-2")

	require.ErrorContains(t, err, "existing refined package conflicts with provider change")
}
