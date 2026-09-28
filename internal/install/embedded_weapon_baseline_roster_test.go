package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baselineWeaponRosterOptions mirrors exactly the default flag values
// cmd/strategist/plugins_prepare_embedded.go uses (--source
// external-skills-source, --defaults-root internal/embed/defaults, --lock
// external-skills-source.lock.yaml) — the same relative paths recorded
// verbatim into the committed lock's own "source:" field. Matching them
// requires the process CWD to be the repository root, exactly as a
// maintainer running `strategist plugins prepare-embedded` would have it
// (see chdirToRepoRoot).
func baselineWeaponRosterOptions() PrepareEmbeddedOptions {
	return PrepareEmbeddedOptions{
		Source:       "external-skills-source",
		DefaultsRoot: filepath.Join("internal", "embed", "defaults"),
		LockPath:     EmbeddedSkillLockFileName,
	}
}

// chdirToRepoRoot changes the process CWD from this package's directory
// (internal/install) to the repository root for the duration of the test,
// restoring it on cleanup. Not safe to combine with t.Parallel() — CWD is
// process-global.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repoRoot))
	t.Cleanup(func() { require.NoError(t, os.Chdir(orig)) })
}

// TestEmbeddedSkillBaselineRoster_AllThreeIngest guards the requester's
// explicit item-3 baseline: brainstorming, openspec-explore, and
// openspec-propose are a permanent roster under external-skills-source/ —
// they may be updated in place but must always resolve and ingest cleanly
// through the real pipeline (trust/id-shadowing/dependency checks), never
// rejected, independent of whether the committed catalog/lock currently have
// drifted (see TestEmbeddedSkillBaselineRoster_NoDrift below for that
// separate guarantee).
func TestEmbeddedSkillBaselineRoster_AllThreeIngest(t *testing.T) {
	chdirToRepoRoot(t)

	report, _, err := CheckEmbeddedDrift(baselineWeaponRosterOptions())
	require.NoError(t, err)
	require.Emptyf(t, report.Rejected, "no baseline package should be rejected by real ingestion: %+v", report.Rejected)

	ingestedIDs := make(map[string]bool, len(report.Ingested))
	for _, skill := range report.Ingested {
		ingestedIDs[skill.ID] = true
	}
	for _, want := range []string{"brainstorming", "openspec-explore", "openspec-propose"} {
		assert.Truef(t, ingestedIDs[want], "expected %q to be ingested from external-skills-source/", want)
	}
}

// TestEmbeddedSkillBaselineRoster_NoDrift fails the build if
// external-skills-source/'s baseline roster (or any other package placed
// there) has diverged from the committed
// internal/embed/defaults/plugins/catalog.yaml, its skill.yaml mirrors, or
// external-skills-source.lock.yaml — i.e. if `strategist plugins
// prepare-embedded` was not re-run and committed after a source change. This
// closes the gap where that drift was previously only caught by a manual,
// human-run `--check` CI step (cmd/strategist/plugins_prepare_embedded.go)
// with no `go test` coverage at all.
func TestEmbeddedSkillBaselineRoster_NoDrift(t *testing.T) {
	chdirToRepoRoot(t)

	_, drift, err := CheckEmbeddedDrift(baselineWeaponRosterOptions())
	require.NoError(t, err)
	assert.False(t, drift, "external-skills-source/ has drifted from the committed catalog/mirrors/lock — "+
		"run `strategist plugins prepare-embedded` and commit the result")
}
