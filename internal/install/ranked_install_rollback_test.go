package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
	"github.com/stretchr/testify/require"
)

const rankedWizardInput = "en\nen\nen\nen\nepic\n.analysis\nbrainstorming::ranked\nopenspec-propose::ranked\nsniper::ranked\n\n"

// withoutHostOpenSpec removes Node from PATH, so the Ranked bootstrap hits
// exactly the missing-executable failure a clean standalone client sees.
// It uses no fake executable, so it runs on Windows too.
func withoutHostOpenSpec(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// registerFakeHostNode provides deterministic host prerequisites for installer
// tests without putting Node or OpenSpec on the developer machine's PATH.
func registerFakeHostNode(t *testing.T, workspace string) {
	t.Helper()
	originalFind, originalVersion, originalRun := findHostNode, hostNodeVersion, runRankedRuntimeCommand
	t.Cleanup(func() {
		findHostNode, hostNodeVersion, runRankedRuntimeCommand = originalFind, originalVersion, originalRun
	})
	findHostNode = func(string) (string, error) { return filepath.Join(workspace, "fake-node"), nil }
	hostNodeVersion = func(context.Context, string, string) ([]byte, error) { return []byte("v20.19.0\n"), nil }
	runRankedRuntimeCommand = fakeRankedRuntimeCommand(workspace)
}

// fakeRankedRuntimeCommand stands in for the OpenSpec CLI: `init` scaffolds the
// config file, and every command answers with a minimal status document.
func fakeRankedRuntimeCommand(workspace string) func(context.Context, string, string, ...string) ([]byte, error) {
	status := []byte(`{"root":{"path":"` + filepath.ToSlash(filepath.Join(workspace, ".strategist")) + `"},"status":[]}`)
	return func(_ context.Context, commandDir, _ string, args ...string) ([]byte, error) {
		if isRankedInitCommand(args) {
			if err := fakeOpenSpecInit(commandDir); err != nil {
				return nil, err
			}
		}
		return status, nil
	}
}

func isRankedInitCommand(args []string) bool {
	if len(args) > 0 && strings.HasSuffix(args[0], ".mjs") {
		args = args[1:]
	}
	return len(args) > 0 && args[0] == "init"
}

func fakeOpenSpecInit(commandDir string) error {
	if err := os.MkdirAll(filepath.Join(commandDir, "openspec"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(commandDir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644)
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
