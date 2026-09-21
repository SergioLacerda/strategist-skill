package install

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// silentRankedExtractor is the ranked wizard fixture plus an epic template that
// names real slot providers, as the embedded template does.
type silentRankedExtractor struct{ rankedWizardExtractor }

func (silentRankedExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == epicStandaloneTemplatePath {
		return []byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\n"), nil
	}
	return rankedWizardExtractor{}.ReadFile(relPath)
}

func silentInstall(t *testing.T, dir string) error {
	t.Helper()
	svc := Service{Extractor: silentRankedExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: filepath.Join(dir, "shim-home")}
	return svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, NoShim: true})
}

func bindingModes(t *testing.T, dir string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".strategist", "plugins.lock"))
	require.NoError(t, err)
	var lock domain.PluginLockFile
	require.NoError(t, yaml.Unmarshal(raw, &lock))
	modes := map[string]string{}
	for _, b := range lock.Bindings {
		modes[b.Slot] = b.EffectiveMode()
	}
	return modes
}

// With an embedded payload the silent install must reach the same ranked
// runtime state as the wizard install: nothing is left green but unrunnable.
func TestSilentInstall_WithEmbeddedPayloadMaterializesRankedRuntime(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	registerFakePayload(t)
	t.Setenv("PATH", t.TempDir())

	require.NoError(t, silentInstall(t, dir))

	require.Equal(t, domain.SlotBindingModeRanked, bindingModes(t, dir)["refinement"])
	require.FileExists(t, filepath.Join(dir, ".strategist", "openspec", "config.yaml"))
	raw, err := os.ReadFile(filepath.Join(dir, ".strategist", rankedRuntimeStatePath))
	require.NoError(t, err)
	var state rankedRuntimeState
	require.NoError(t, json.Unmarshal(raw, &state))
	require.Len(t, state.Entries, 1)
	require.Equal(t, "refinement", state.Entries[0].Slot)
	require.NotNil(t, state.Entries[0].Runtime)
}

// Slots whose provider has no runtime keep their custom binding: the silent
// install only promotes what needs a runtime.
func TestSilentInstall_PromotesOnlyProvidersThatDeclareARuntime(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	registerFakePayload(t)
	t.Setenv("PATH", t.TempDir())

	require.NoError(t, silentInstall(t, dir))

	modes := bindingModes(t, dir)
	require.NotEqual(t, domain.SlotBindingModeRanked, modes["discovery"])
}

// Without a payload the silent install stays as it was (custom) and does not
// invent a runtime; the readiness gate is what reports the missing executable.
func TestSilentInstall_WithoutPayloadStaysCustomAndMaterializesNothing(t *testing.T) {
	dir := t.TempDir()
	withoutHostOpenSpec(t)
	original := payloadSource
	t.Cleanup(func() { payloadSource = original })
	payloadSource = func() (runtimepayload.Embedded, bool) { return runtimepayload.Embedded{}, false }

	require.NoError(t, silentInstall(t, dir))

	require.NotEqual(t, domain.SlotBindingModeRanked, bindingModes(t, dir)["refinement"])
	require.NoDirExists(t, filepath.Join(dir, ".strategist", "weapon-runtime"))
}
