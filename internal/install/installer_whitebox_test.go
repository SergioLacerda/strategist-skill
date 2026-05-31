package install

// Whitebox tests for install — covers error paths and wizard integration
// that cannot be reached through the public API without injected stdin.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalExtractor creates the minimum .strategist/ layout needed by Install.
type minimalExtractor struct{}

func (m minimalExtractor) Extract(targetDir string, _ bool) error {
	dirs := []string{
		filepath.Join(targetDir, "personas"),
		filepath.Join(targetDir, "roles"),
		filepath.Join(targetDir, "templates"),
		filepath.Join(targetDir, "memory"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	files := map[string]string{
		filepath.Join(targetDir, "active.yaml"):                            "mode: full\n",
		filepath.Join(targetDir, "SKILL.md"):                               "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                   "sources: []\n",
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

type nopCompiler struct{}

func (nopCompiler) CompileAll(_, _ string) error { return nil }

func newSvcW(wizardInput string) Service {
	return Service{
		Extractor:    minimalExtractor{},
		Compiler:     nopCompiler{},
		WizardReader: strings.NewReader(wizardInput),
	}
}

// --- Wizard path ---

func TestInstall_WizardPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// lang_ui, mode, base_path, language, adr, discovery, refinement, execution, chest
	svc := newSvcW("en\nminimal\n/workspace\npt\nyes\nbrainstorming\narchivist\nsdd-ask\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, readErr)
	s := string(data)
	assert.Contains(t, s, "mode: minimal")
	assert.Contains(t, s, "base_path: /workspace")
	assert.Contains(t, s, "language: pt")
	assert.Contains(t, s, "adr_enabled: true")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sdd-ask")
	assert.NotContains(t, s, "roles_config")
}

func TestInstall_WizardPath_Defaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW("\n\n\n\n\n\n\n\n\n") // all defaults (9 prompts)
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	s := string(data)
	assert.Contains(t, s, "mode: full")
	assert.Contains(t, s, "language: pt")
	assert.Contains(t, s, "adr_enabled: true")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: openspec-explore")
	assert.Contains(t, s, "execution: sdd-ask")
}

// --- copyTemplate error path ---

func TestInstall_CopyTemplateMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Extractor that does NOT create the template file
	svc := Service{
		Extractor: &noTemplateExtractor{},
		Compiler:  nopCompiler{},
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "copy template")
}

type noTemplateExtractor struct{}

func (n *noTemplateExtractor) Extract(targetDir string, _ bool) error {
	return os.MkdirAll(targetDir, 0o755)
}

// --- ensureGitignore: no trailing newline edge case ---

func TestEnsureGitignore_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	// Write existing content WITHOUT trailing newline
	require.NoError(t, os.WriteFile(gi, []byte("*.log"), 0o644))
	require.NoError(t, ensureGitignore(dir))
	data, err := os.ReadFile(gi)
	require.NoError(t, err)
	assert.Contains(t, string(data), ".strategist/.compiled/")
}

func TestEnsureGitignore_AlreadyPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte(".strategist/.compiled/\n"), 0o644))
	require.NoError(t, ensureGitignore(dir))
	data, err := os.ReadFile(gi)
	require.NoError(t, err)
	// Must not duplicate the entry
	assert.Equal(t, 1, strings.Count(string(data), ".strategist/.compiled/"))
}

func TestEnsureGitignore_OpenError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	// .gitignore does not exist; make parent dir read-only so OpenFile fails.
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := ensureGitignore(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "open .gitignore")
}

func TestEnsureGitignore_ReadError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte(""), 0o000))
	t.Cleanup(func() { _ = os.Chmod(gi, 0o644) })
	err := ensureGitignore(dir)
	require.Error(t, err)
}

// --- writeActiveYAML error path ---

func TestWriteActiveYAML_ReadOnlyDir(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := writeActiveYAML(dir, domain.WizardConfig{
		Mode: "full", BasePath: ".", UILanguage: "pt", DocLanguage: "pt", ChatLanguage: "pt", CodeLanguage: "pt", AdrEnabled: true,
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-explore", ExecutionProvider: "sdd-ask",
	})
	require.Error(t, err)
}

// --- copyTemplate error paths ---

func TestCopyTemplate_MissingSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := copyTemplate(dir, "nonexistent/template.yaml", "active.yaml")
	require.Error(t, err)
	assert.ErrorContains(t, err, "read template")
}

func TestCopyTemplate_WriteError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create template source
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "src.yaml"), []byte("x: 1\n"), 0o644))
	// Make the destination a directory so os.WriteFile to it fails (EISDIR)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "active.yaml"), 0o755))
	err := copyTemplate(dir, "templates/src.yaml", "active.yaml")
	require.Error(t, err)
	assert.ErrorContains(t, err, "write")
}

func TestRunWizard_EOFOnFirstPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader(""))
	require.Error(t, err)
	assert.ErrorContains(t, err, "language_ui")
}

func TestRunWizard_EOFOnSecondPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "mode")
}

func TestRunWizard_EOFOnThirdPrompt_BasePath(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "base_path")
}

func TestRunWizard_EOFOnFourthPrompt_Language(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n.\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "language")
}

func TestRunWizard_EOFOnFifthPrompt_Adr(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n.\npt\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "adr_enabled")
}

func TestRunWizard_EOFOnSixthPrompt_Discovery(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n.\npt\nyes\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "discovery")
}

func TestRunWizard_EOFOnSeventhPrompt_Refinement(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n.\npt\nyes\nbrainstorming\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "refinement")
}

func TestRunWizard_EOFOnEighthPrompt_Execution(t *testing.T) {
	t.Parallel()
	_, err := runWizard(strings.NewReader("en\nfull\n.\npt\nyes\nbrainstorming\nopenspec-explore\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "execution")
}

// --- installShimTo error paths ---

func TestInstallShimTo_ReadOnlyParent(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	home := t.TempDir()
	require.NoError(t, os.Chmod(home, 0o444))
	t.Cleanup(func() { _ = os.Chmod(home, 0o755) })
	err := installShimTo(home)
	require.Error(t, err)
	assert.ErrorContains(t, err, "mkdir shim dir")
}

func TestInstallShimTo_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	home := t.TempDir()
	shimDir := filepath.Join(home, ".claude", "skills", "strategist")
	require.NoError(t, os.MkdirAll(shimDir, 0o755))
	// Make SKILL.md a directory so WriteFile to it fails (EISDIR)
	require.NoError(t, os.Mkdir(filepath.Join(shimDir, "SKILL.md"), 0o755))
	err := installShimTo(home)
	require.Error(t, err)
	assert.ErrorContains(t, err, "write shim")
}

// --- Install: error propagation for gitignore and shim ---

func TestInstall_ShimError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	shimHome := t.TempDir()
	require.NoError(t, os.Chmod(shimHome, 0o444))
	t.Cleanup(func() { _ = os.Chmod(shimHome, 0o755) })

	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: shimHome,
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "shim")
}

func TestInstall_GitignoreError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	// Write unreadable .gitignore to trigger a stat/read error
	require.NoError(t, os.WriteFile(gi, []byte(""), 0o000))
	t.Cleanup(func() { _ = os.Chmod(gi, 0o644) })

	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "gitignore")
}
