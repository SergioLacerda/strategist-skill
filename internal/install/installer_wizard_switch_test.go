package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// switchWizardExtractor is rankedWizardExtractor with a second certified
// Archivist: the embedded prompt-bridge writing-plans (ADR-0067), selectable
// alongside openspec-propose.
type switchWizardExtractor struct{ rankedWizardExtractor }

const writingPlansWizardCatalogEntry = `  - id: writing-plans
    version: 1.0.0
    risk_score: write_analysis
    canonical_role: archivist
    ranked: true
    certification_digest: sha256:writing-plans
    ranked_binding_generation: 1
    ranked_binding_status: active
    runtime:
      kind: embedded
      execution_mode: prompt_bridge
    installable: true
    legacy_manifest_path: skills/writing-plans/skill.yaml
    compatibility_source: embedded
`

func switchWizardCatalog() string {
	return rankedWizardCatalogYAML + writingPlansWizardCatalogEntry
}

func (e switchWizardExtractor) Extract(targetDir string, withShim bool) error {
	if err := e.rankedWizardExtractor.Extract(targetDir, withShim); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(targetDir, pluginCatalogPath), []byte(switchWizardCatalog()), 0o644)
}

func (switchWizardExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == pluginCatalogPath {
		return []byte(switchWizardCatalog()), nil
	}
	return rankedWizardExtractor{}.ReadFile(relPath)
}

func wizardInputChoosingRefinement(choice string) string {
	return "en\nen\nen\nen\nepic\n.analysis\n\n\nbrainstorming::ranked\n" + choice + "\nsniper::ranked\n\n"
}

func installWithRefinementChoice(t *testing.T, dir, choice string) error {
	t.Helper()
	svc := Service{
		Extractor: switchWizardExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir(),
		WizardPrompter: NewTextPrompter(strings.NewReader(wizardInputChoosingRefinement(choice))),
	}
	return svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
}

func requireRefinement(t *testing.T, dir, provider, runtimeKind string) {
	t.Helper()
	active, err := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(active), "refinement: "+provider)
	lock, err := readPluginLockFile(filepath.Join(dir, ".strategist"))
	require.NoError(t, err)
	binding, ok := findSlotBinding(lock.Bindings, "refinement")
	require.True(t, ok)
	assert.Equal(t, provider, binding.InstalledInstanceID)
	assert.Equal(t, domain.SlotBindingModeRanked, binding.Mode, "the Ranked choice survives persistence")
	assert.Equal(t, runtimeKind, binding.RuntimeKind)
}

// TestInstall_WizardRerunSwitchesTheRefinementWeaponBetweenCertifiedRankedChoices
// is the regression of a wizard re-run on an existing installation: the
// selection was discarded (install rolled back with
// activation_requires_successful_probe) and `strategist check` kept reporting
// the previous Weapon.
func TestInstall_WizardRerunSwitchesTheRefinementWeaponBetweenCertifiedRankedChoices(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	// No t.Parallel(): t.Setenv modifies the process-global PATH.
	dir := t.TempDir()
	binDir := t.TempDir()
	fakeScript := `#!/bin/sh
set -eu
if [ "$1" = "init" ]; then
  mkdir -p "$PWD/openspec"
  printf 'schema: spec-driven\n' > "$PWD/openspec/config.yaml"
  exit 0
fi
printf '{"root":{"path":"%s"},"members":[],"status":[]}\n' "$(dirname "$PWD")"
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "openspec"), []byte(fakeScript), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	require.NoError(t, installWithRefinementChoice(t, dir, "openspec-propose::ranked"))
	requireRefinement(t, dir, "openspec-propose", domain.RankedRuntimeOpenSpecRoot)

	require.NoError(t, installWithRefinementChoice(t, dir, "writing-plans::ranked"), "switching to the embedded Archivist on an existing install")
	requireRefinement(t, dir, "writing-plans", domain.RankedRuntimeEmbedded)

	require.NoError(t, installWithRefinementChoice(t, dir, "openspec-propose::ranked"), "switching back to the OpenSpec Archivist")
	requireRefinement(t, dir, "openspec-propose", domain.RankedRuntimeOpenSpecRoot)
}
