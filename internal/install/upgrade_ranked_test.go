package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

func readRankedState(t *testing.T, strategist string) domain.RankedRuntimeState {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(strategist, domain.RankedRuntimeStatePath))
	require.NoError(t, err)
	var state domain.RankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	return state
}

// A workspace installed before a certification digest change must be repairable
// by `strategist upgrade`, not only by rerunning the wizard install.
func TestApplyUpgrade_RepairsStaleRankedRuntimeState(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	registerFakeHostNode(t, dir)
	t.Setenv("PATH", t.TempDir())
	svc := Service{Extractor: rankedWizardExtractor{}, Compiler: nopCompiler{}, WizardPrompter: NewTextPrompter(strings.NewReader(rankedWizardInput)), ShimHomeDir: filepath.Join(dir, "shim-home")}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))
	strategist := filepath.Join(dir, ".strategist")
	fresh := readRankedState(t, strategist)
	require.Len(t, fresh.Entries, 1)
	wantDigest := fresh.Entries[0].ContractDigest

	// Simulate the state a previous binary left behind, and a vanished runtime.
	stale := fresh
	stale.Entries[0].ContractDigest = "sha256:stale"
	raw, err := json.Marshal(stale)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(strategist, domain.RankedRuntimeStatePath), raw, 0o644))
	require.NoError(t, os.RemoveAll(filepath.Join(strategist, "weapon-runtime")))

	upgrader := Service{Extractor: rankedWizardExtractor{}, Lister: alwaysAllPathsLister{[]string{"SKILL.md"}}}
	plan, err := upgrader.PlanUpgrade(strategist)
	require.NoError(t, err)
	_, err = upgrader.ApplyUpgrade(strategist, plan, false)
	require.NoError(t, err)

	repaired := readRankedState(t, strategist)
	require.Equal(t, wantDigest, repaired.Entries[0].ContractDigest, "state re-recorded from the installed catalog")
	require.FileExists(t, filepath.Join(strategist, "weapon-runtime", "openspec-propose", "openspec", "dist", "core", "artifact-graph", "openspec.mjs"), "OpenSpec bundle re-materialized")
}

// A record from an earlier release (schema v1, private Node) is rewritten by
// `strategist upgrade` as the current schema with the host Node.
func TestApplyUpgrade_RewritesLegacyRankedRuntimeState(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	registerFakeHostNode(t, dir)
	t.Setenv("PATH", t.TempDir())
	svc := Service{Extractor: rankedWizardExtractor{}, Compiler: nopCompiler{}, WizardPrompter: NewTextPrompter(strings.NewReader(rankedWizardInput)), ShimHomeDir: filepath.Join(dir, "shim-home")}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))
	strategist := filepath.Join(dir, ".strategist")
	require.Equal(t, domain.RankedRuntimeStateSchemaVersion, readRankedState(t, strategist).SchemaVersion)
	legacy := `{"schema_version":"strategist-ranked-runtime/v1","entries":[{"role":"archivist","slot":"refinement","provider":"openspec-propose","contract_digest":"sha256:old","runtime":{"mode":"payload","node":"weapon-runtime/openspec-propose/node/bin/node","script":"weapon-runtime/openspec-propose/openspec/x.mjs"}}]}`
	require.NoError(t, os.WriteFile(filepath.Join(strategist, domain.RankedRuntimeStatePath), []byte(legacy), 0o644))

	upgrader := Service{Extractor: rankedWizardExtractor{}, Lister: alwaysAllPathsLister{[]string{"SKILL.md"}}}
	plan, err := upgrader.PlanUpgrade(strategist)
	require.NoError(t, err)
	_, err = upgrader.ApplyUpgrade(strategist, plan, false)
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(strategist, domain.RankedRuntimeStatePath))
	require.NoError(t, err)
	state, err := domain.ParseRankedRuntimeState(raw)
	require.NoError(t, err, "upgrade must leave a current-schema record")
	entry, ok := state.Entry("refinement", "openspec-propose")
	require.True(t, ok)
	require.True(t, filepath.IsAbs(entry.Runtime.Node), "host Node is recorded as an absolute path")
	require.NotContains(t, string(raw), `"mode"`)
}

// A workspace with no ranked binding is untouched by the reconciliation.
func TestApplyUpgrade_WithoutRankedBindingsWritesNoRuntimeState(t *testing.T) {
	dir := t.TempDir()
	registerFakeHostNode(t, dir)
	svc := Service{Extractor: silentRankedExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: filepath.Join(dir, "shim-home"), Lister: alwaysAllPathsLister{[]string{"SKILL.md", "roles/default.yaml", pluginCatalogPath}}}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, NoShim: true}))
	strategist := filepath.Join(dir, ".strategist")
	// Remove the active lock to model a workspace with no Ranked binding before
	// running upgrade reconciliation.
	require.NoError(t, os.Remove(filepath.Join(strategist, "plugins.lock")))
	require.NoError(t, os.Remove(filepath.Join(strategist, domain.RankedRuntimeStatePath)))

	plan, err := svc.PlanUpgrade(strategist)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(strategist, plan, false)
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(strategist, domain.RankedRuntimeStatePath))
}
