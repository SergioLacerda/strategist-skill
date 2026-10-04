package main

// Composition dependency regression coverage.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// workspaceWithRoot returns a project dir holding a minimal .strategist root,
// and makes it the current directory.
func workspaceWithRoot(t *testing.T) (project, root string) {
	t.Helper()
	project = t.TempDir()
	root = filepath.Join(project, ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	testutil.MinimalRoot(t, root)
	t.Chdir(project)
	return project, root
}

func TestFilterMissionLevelRecords(t *testing.T) {
	records := []leveling.Record{{MissionID: "a"}, {MissionID: "b"}, {MissionID: "a"}}
	assert.Len(t, filterMissionLevelRecords(records, "a"), 2)
	assert.Empty(t, filterMissionLevelRecords(records, "none"))
}

func TestSilenceRunClosuresHandleBothContexts(t *testing.T) {
	plain := &cobra.Command{}
	withRun := &cobra.Command{}
	attachMissionRun(t, withRun)
	for _, silence := range []func(*cobra.Command){evalDependencies().SilenceRun, evalHarvestDependencies().SilenceRun, metricsDependencies().SilenceRun} {
		silence(plain)
		silence(withRun)
	}
}

func TestResolveRootsRejectABrokenExplicitRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	cmd := &cobra.Command{}
	_, _, err := resolveEvalActionRoot(cmd, "harvest", "")
	require.ErrorContains(t, err, "eval harvest:")
	_, err = resolveMetricsRoot(cmd, "show", "")
	require.ErrorContains(t, err, "metrics show:")

	_, root := workspaceWithRoot(t)
	got, _, err := resolveEvalActionRoot(cmd, "harvest", root)
	require.NoError(t, err)
	assert.Equal(t, root, got)
	got, err = resolveMetricsRoot(cmd, "show", "")
	require.NoError(t, err)
	assert.Equal(t, root, got)
}

func TestResolveMetricsBasePath(t *testing.T) {
	bare := t.TempDir()
	base, err := resolveMetricsBasePath(bare)
	require.NoError(t, err)
	assert.Empty(t, base, "a runtime without active.yaml has no artifact tree")

	_, root := workspaceWithRoot(t)
	base, err = resolveMetricsBasePath(root)
	require.NoError(t, err)
	assert.NotEmpty(t, base)

	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("base_path: \"\"\n"), 0o644))
	_, err = resolveMetricsBasePath(root)
	require.ErrorContains(t, err, "resolve active base path")
}

func TestMechanismsBriefFailureModes(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.ErrorContains(t, runMechanismsBrief(cmd, "", "not-a-role", false), "is not a Strategist role")

	missing := filepath.Join(t.TempDir(), "absent", ".strategist")
	require.ErrorContains(t, runMechanismsBrief(cmd, missing, "ranger", false), "mechanisms brief:")

	_, root := workspaceWithRoot(t)
	err := runMechanismsBrief(cmd, root, "ranger", true)
	if err != nil {
		assert.Contains(t, err.Error(), "mechanisms brief:")
	}
}

func TestLevelingDependencyFailureModes(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := levelingWorkspaceRoot()
	require.ErrorContains(t, err, "leveling:")
	_, _, err = loadLevelingPolicy()
	require.Error(t, err)

	_, root := workspaceWithRoot(t)
	cfg, warning := readActiveLevelingConfig(root)
	assert.Empty(t, warning)
	assert.Equal(t, domain.LevelingConfig{}, cfg)

	_, warning = readActiveLevelingConfig(t.TempDir())
	assert.Empty(t, warning, "a missing active.yaml degrades silently")

	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("leveling: [unclosed"), 0o644))
	_, warning = readActiveLevelingConfig(root)
	assert.NotEmpty(t, warning)

	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("leveling:\n  mode: bogus\n"), 0o644))
	_, warning = readActiveLevelingConfig(root)
	assert.NotEmpty(t, warning)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "broken.yaml"), []byte("role: [unclosed"), 0o644))
	reg, warning := loadRoleRegistry(root)
	assert.NotEmpty(t, warning)
	assert.True(t, reg.Has("ranger"), "a broken role file degrades to the built-ins")

	emitRoleLevel(context.Background(), "m1", "run-1", leveling.Level{Role: "ranger", Provider: "p", Capability: "c", PolicyVersion: 2, PolicyDigest: "d"}, "reason")
	emitRoleLevel(context.Background(), "m1", "", leveling.Level{Role: "ranger"}, "")
}

func TestNormalizeHelpersRejectABrokenRoot(t *testing.T) {
	broken := missionadapter.NormalizeOptions{Root: filepath.Join(t.TempDir(), "absent", ".strategist"), MissionID: "m1"}
	_, _, _, err := resolveNormalizePaths(broken)
	require.ErrorContains(t, err, "resolve active base path")
	require.ErrorContains(t, recordNormalizeConfidence(broken, domain.ConfidenceClaim{}, nil), "resolve active base path for confidence")
	_, err = resolveNormalizeGateLabel(broken)
	require.ErrorContains(t, err, "resolve active base path")

	_, root := workspaceWithRoot(t)
	good := missionadapter.NormalizeOptions{Root: root, MissionID: "m1", RuntimeRoot: "rt", Pending: "p.md"}
	basePath, runtimeRoot, pending, err := resolveNormalizePaths(good)
	require.NoError(t, err)
	assert.NotEmpty(t, basePath)
	assert.Equal(t, filepath.Join(filepath.Dir(root), "rt"), runtimeRoot)
	assert.Equal(t, filepath.Join(filepath.Dir(root), "p.md"), pending)
	abs := filepath.Join(t.TempDir(), "abs.md")
	assert.Equal(t, abs, resolvePath(abs, "fallback", "/project"))
	assert.Equal(t, "fallback", resolvePath("", "fallback", "/project"))

	_, err = resolveNormalizeGateLabel(missionadapter.NormalizeOptions{Root: root, MissionID: "m1"})
	if err != nil {
		assert.Contains(t, err.Error(), "gate outcome")
	}
	require.Error(t, recordNormalizeConfidence(good, domain.ConfidenceClaim{}, nil), "an empty claim is not recordable")
}
