package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/stretchr/testify/require"
)

const rankedWizardInput = "en\nen\nen\nen\nepic\n.analysis\nbrainstorming::ranked\nopenspec-propose::ranked\nsniper::ranked\n\n"

// withoutHostOpenSpec removes any embedded payload and any openspec from PATH,
// so the Ranked bootstrap hits exactly the missing-executable failure a clean
// standalone client sees. It uses no fake executable, so it runs on Windows too.
func withoutHostOpenSpec(t *testing.T) {
	t.Helper()
	original := payloadSource
	t.Cleanup(func() { payloadSource = original })
	payloadSource = func() (runtimepayload.Embedded, bool) { return runtimepayload.Embedded{}, false }
	t.Setenv("PATH", t.TempDir())
}

func rankedInstall(dir string) error {
	svc := Service{Extractor: rankedWizardExtractor{}, Compiler: nopCompiler{}, WizardPrompter: NewTextPrompter(strings.NewReader(rankedWizardInput)), ShimHomeDir: filepath.Join(dir, "shim-home")}
	return svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
}

// A Ranked install that cannot find its executable must fail with the
// cataloged diagnostic and leave a fresh workspace exactly as it found it.
func TestInstall_RankedMissingExecutableRollsBackFreshWorkspace(t *testing.T) {
	dir := t.TempDir()
	withoutHostOpenSpec(t)

	err := rankedInstall(dir)

	require.Error(t, err)
	require.ErrorContains(t, err, domain.ReasonRankedRuntimeExecutableMissing)
	require.ErrorContains(t, err, `"openspec-propose"`)
	var missing *runtimeenv.ExecutableNotFoundError
	require.ErrorAs(t, err, &missing)
	require.NoDirExists(t, filepath.Join(dir, ".strategist"), "a fresh install must be fully rolled back")
}

// Reinstalling over an existing tree must keep what was there and leave none
// of the ranked runtime artifacts behind.
func TestInstall_RankedMissingExecutableKeepsPreexistingContent(t *testing.T) {
	dir := t.TempDir()
	strategist := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategist, 0o755))
	sentinel := filepath.Join(strategist, "SENTINEL")
	require.NoError(t, os.WriteFile(sentinel, []byte("keep"), 0o644))
	withoutHostOpenSpec(t)

	err := rankedInstall(dir)

	require.ErrorContains(t, err, domain.ReasonRankedRuntimeExecutableMissing)
	raw, readErr := os.ReadFile(sentinel)
	require.NoError(t, readErr)
	require.Equal(t, "keep", string(raw))
	for _, artifact := range []string{"ranked-runtimes.yaml", "weapon-runtime", "openspec"} {
		require.NoFileExists(t, filepath.Join(strategist, artifact), artifact)
	}
}
