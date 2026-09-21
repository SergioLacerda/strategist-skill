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

func readRankedState(t *testing.T, strategist string) rankedRuntimeState {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(strategist, rankedRuntimeStatePath))
	require.NoError(t, err)
	var state rankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	return state
}

// A workspace installed before a certification digest change must be repairable
// by `strategist upgrade`, not only by rerunning the wizard install.
func TestApplyUpgrade_RepairsStaleRankedRuntimeState(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	registerFakePayload(t)
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
	require.NoError(t, os.WriteFile(filepath.Join(strategist, rankedRuntimeStatePath), raw, 0o644))
	require.NoError(t, os.RemoveAll(filepath.Join(strategist, "weapon-runtime")))

	upgrader := Service{Extractor: rankedWizardExtractor{}, Lister: alwaysAllPathsLister{[]string{"SKILL.md"}}}
	plan, err := upgrader.PlanUpgrade(strategist)
	require.NoError(t, err)
	_, err = upgrader.ApplyUpgrade(strategist, plan, false)
	require.NoError(t, err)

	repaired := readRankedState(t, strategist)
	require.Equal(t, wantDigest, repaired.Entries[0].ContractDigest, "state re-recorded from the installed catalog")
	require.FileExists(t, filepath.Join(strategist, "weapon-runtime", "openspec-propose", "node", "bin", "node"), "runtime re-materialized")
}

// A workspace with no ranked binding is untouched by the reconciliation.
func TestApplyUpgrade_WithoutRankedBindingsWritesNoRuntimeState(t *testing.T) {
	dir := t.TempDir()
	withoutHostOpenSpec(t)
	svc := Service{Extractor: silentRankedExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: filepath.Join(dir, "shim-home"), Lister: alwaysAllPathsLister{[]string{"SKILL.md"}}}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, NoShim: true}))
	strategist := filepath.Join(dir, ".strategist")

	plan, err := svc.PlanUpgrade(strategist)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(strategist, plan, false)
	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(strategist, rankedRuntimeStatePath))
}
