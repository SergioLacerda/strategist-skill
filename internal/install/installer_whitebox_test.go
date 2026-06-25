package install

// Whitebox tests for install — covers error paths and wizard integration
// that cannot be reached through the public API without injected stdin.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
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
		filepath.Join(targetDir, "SKILL.md"):                               "# SKILL\n",
		filepath.Join(targetDir, "knowledge.index.yaml"):                   "sources: []\n",
		filepath.Join(targetDir, "treasure-chests.yaml"):                   "chests: []\n",
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\nroles_config: roles/default.yaml\n",
		filepath.Join(targetDir, "templates", "epic-standalone.yaml"):      "mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (m minimalExtractor) ReadFile(relPath string) ([]byte, error) {
	switch relPath {
	case "templates/epic-standalone.yaml":
		return []byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\n"), nil
	case "SKILL.md":
		return []byte("# SKILL\n"), nil
	case "skills/brainstorming/skill.yaml":
		return []byte("id: brainstorming\nstatus: active\nrisk_score: write_analysis\nprovider_class: rankeado\nspecialization_taxonomy:\n  canonical_role: ranger\n  provider_class: rankeado\nauxiliary_tools_allowed:\n  - writing-plans\n"), nil
	case "skills/openspec-explore/skill.yaml":
		return []byte("id: openspec-explore\nstatus: active\nrisk_score: write_analysis\n"), nil
	default:
		return nil, fmt.Errorf("minimalExtractor: file not found: %s", relPath)
	}
}

type nopCompiler struct{}

func (nopCompiler) CompileAll(_, _ string) error { return nil }

func newSvcW(t *testing.T, wizardInput string) Service {
	t.Helper()
	return Service{
		Extractor:      minimalExtractor{},
		Compiler:       nopCompiler{},
		WizardPrompter: NewTextPrompter(strings.NewReader(wizardInput)),
		ShimHomeDir:    t.TempDir(),
	}
}

// --- Wizard path ---

func TestInstall_WizardPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 11 prompts: ui/doc/chat/code/mode/base/adr/discovery/refinement/execution/chest
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nyes\nbrainstorming\narchivist\nsdd-ask\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, readErr)
	s := string(data)
	assert.Contains(t, s, "mode: epic")
	assert.Contains(t, s, "base_path: /workspace")
	assert.Contains(t, s, "roles_config: roles/default.yaml")
	assert.Contains(t, s, "ui: en")
	assert.Contains(t, s, "docs: en")
	assert.Contains(t, s, "chat: pt-BR")
	assert.Contains(t, s, "code: en")
	assert.Contains(t, s, "adr_enabled: true")
	assert.NotContains(t, s, "execution_mode")
	assert.NotContains(t, s, "git_persistence_mode")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sdd-ask")

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestInstall_WizardPath_WithChest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 11 prompts (no execution_mode/git_persistence_mode)
	svc := newSvcW(t, "en\nen\nen\nen\npragmatic\n.analysis\nyes\nbrainstorming\nopenspec-explore\nsdd-ask\n.sdd/source\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	ki, readErr := os.ReadFile(filepath.Join(dir, ".strategist", "knowledge.index.yaml"))
	require.NoError(t, readErr)
	s := string(ki)
	assert.Contains(t, s, "id: source")
	assert.Contains(t, s, "path: .sdd/source")
	assert.Contains(t, s, "tags: [all]")
	assert.NotContains(t, s, "sources: []")
}

func TestInstall_WizardPath_Defaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW(t, "\n\n\n\n\n\n\n\n\n\n\n") // all defaults (11 prompts)
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	s := string(data)
	assert.Contains(t, s, "mode: epic")
	assert.Contains(t, s, "roles_config: roles/default.yaml")
	assert.Contains(t, s, "ui: en")
	assert.Contains(t, s, "docs: en")
	assert.Contains(t, s, "chat: en")
	assert.Contains(t, s, "code: en")
	assert.Contains(t, s, "adr_enabled: true")
	assert.NotContains(t, s, "execution_mode")
	assert.NotContains(t, s, "git_persistence_mode")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: openspec-explore")
	assert.Contains(t, s, "execution: sdd-ask")

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "id: brainstorming")
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	openspecExplore, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(openspecExplore), "id: openspec-explore")
	assert.Contains(t, string(openspecExplore), "risk_score: write_analysis")
}

func TestInstall_WizardPath_ExplicitDefaultProvidersMaterializeManifests(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\nen\nen\nepic\n.analysis\nyes\nbrainstorming\nopenspec-explore\nsdd-ask\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	openspecExplore, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(openspecExplore), "risk_score: write_analysis")
}

// --- applyConfig: active.yaml preservation and force-overwrite ---

func TestApplyConfig_PreservesExistingActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
	}
	// First install writes active.yaml from embedded template.
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	// Simulate user customization.
	customContent := "mode: epic\nbase_path: .custom\n"
	activeYAMLPath := filepath.Join(dir, ".strategist", "active.yaml")
	require.NoError(t, os.WriteFile(activeYAMLPath, []byte(customContent), 0o644))

	// Re-install without --force must preserve the customized file.
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	got, err := os.ReadFile(activeYAMLPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, string(got), "re-install must not overwrite user-customized active.yaml")
}

func TestApplyConfig_ForceOverwritesActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
	}
	// First install.
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	// Simulate user customization.
	activeYAMLPath := filepath.Join(dir, ".strategist", "active.yaml")
	require.NoError(t, os.WriteFile(activeYAMLPath, []byte("mode: epic\nbase_path: .custom\n"), 0o644))

	// Re-install with --force must overwrite with the embedded template.
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true}))

	got, err := os.ReadFile(activeYAMLPath)
	require.NoError(t, err)
	assert.Equal(t, "mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\n", string(got), "--force must overwrite active.yaml with embedded template")
}

func TestApplyConfig_ReadFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   &errReadExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "read template")
}

// errReadExtractor creates the .strategist/ dir but returns an error from ReadFile.
type errReadExtractor struct{}

func (e *errReadExtractor) Extract(targetDir string, _ bool) error {
	return os.MkdirAll(targetDir, 0o755)
}

func (e *errReadExtractor) ReadFile(relPath string) ([]byte, error) {
	return nil, fmt.Errorf("errReadExtractor: read error for %s", relPath)
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
		Mode: "pragmatic", BasePath: ".", UILanguage: "pt", DocLanguage: "pt", ChatLanguage: "pt", CodeLanguage: "pt", AdrEnabled: true,
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-explore", ExecutionProvider: "sdd-ask",
	})
	require.Error(t, err)
}

func p(input string) Prompter { return NewTextPrompter(strings.NewReader(input)) }

func TestRunWizard_EOFOnFirstPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p(""), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "ui_language")
}

func TestRunWizard_EOFOnSecondPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "doc_language")
}

func TestRunWizard_EOFOnThirdPrompt_ChatLang(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "chat_language")
}

func TestRunWizard_EOFOnFourthPrompt_CodeLang(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "code_language")
}

func TestRunWizard_EOFOnFifthPrompt_Mode(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "mode")
}

func TestRunWizard_EOFOnSixthPrompt_BasePath(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "base_path")
}

func TestRunWizard_EOFOnSeventhPrompt_Adr(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "adr_enabled")
}

func TestRunWizard_EOFOnEighthPrompt_Discovery(t *testing.T) {
	t.Parallel()
	// 8th prompt is now discovery (no more execution_mode/git_persistence_mode steps)
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "discovery")
}

func TestRunWizard_EOFOnNinthPrompt_Refinement(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nbrainstorming\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "refinement")
}

func TestRunWizard_EOFOnTenthPrompt_Execution(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nbrainstorming\nopenspec-explore\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "execution")
}

func TestRunWizard_EOFOnEleventhPrompt_ChestPath(t *testing.T) {
	t.Parallel()
	// All prompts answered except chest path (11th)
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nbrainstorming\nopenspec-explore\nsdd-ask\n"), minimalExtractor{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "treasure_chest")
}

// --- applyConfig: nil WizardPrompter with non-TTY detection ---

func TestApplyConfig_NilPrompter_NonTTY_FailsOnEmptyStdin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:        minimalExtractor{},
		Compiler:         nopCompiler{},
		ShimHomeDir:      t.TempDir(),
		terminalDetector: func() bool { return false }, // force non-TTY path
		stdinReader:      strings.NewReader(""),        // empty stdin → EOF immediately
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "wizard")
}

func TestApplyConfig_NilPrompter_TTY_FailsOnNoTerminal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:        minimalExtractor{},
		Compiler:         nopCompiler{},
		ShimHomeDir:      t.TempDir(),
		terminalDetector: func() bool { return true }, // force TTY path
		// Use errRun so the test is deterministic and never blocks on a real or open-pipe stdin.
		tuiPrompterFn: func() Prompter { return &TUIPrompter{runFn: errRun} },
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "wizard")
}

func TestApplyConfig_NilPrompter_NilDetector_UsesRealTTYCheck(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("skipped in interactive terminal: real TTY detection would route to TUIPrompter which blocks")
	}
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
		stdinReader: strings.NewReader(""), // empty stdin → EOF immediately
		// Both WizardPrompter and terminalDetector are nil — exercises the real
		// term.IsTerminal path. In CI stdin is not a TTY, so it falls through to
		// NewTextPrompter(stdinReader) and hits EOF immediately.
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "wizard")
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
	err := installShimTo(home, "", "")
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
	err := installShimTo(home, "", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "write shim")
}

func TestReadLocalSKILLMD_ReadFileFails(t *testing.T) {
	t.Parallel()
	svc := Service{
		Extractor:   &errReadExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
	}
	_, err := svc.readLocalSKILLMD(context.Background(), t.TempDir())
	require.Error(t, err)
	assert.ErrorContains(t, err, "read embedded SKILL.md")
}

// --- Install: error propagation for gitignore and shim ---

func TestInstall_ShimError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	shimHome := t.TempDir()

	// Make the .claude parent unwritable so shim installation fails.
	require.NoError(t, os.MkdirAll(filepath.Join(shimHome, ".claude"), 0o755))
	require.NoError(t, os.Chmod(filepath.Join(shimHome, ".claude"), 0o444))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(shimHome, ".claude"), 0o755) })

	svc := Service{
		Extractor:   minimalExtractor{},
		Compiler:    nopCompiler{},
		ShimHomeDir: shimHome,
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "shim")
}

func TestInstallOptionalShims_GeminiAndCodex(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	// Pre-create ~/.gemini/ and ~/.codex/ to trigger optional shim installation.
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))

	installOptionalShims(home, "# SKILL", "")

	expectedPaths := []string{
		filepath.Join(home, ".gemini", "skills", "strategist", "SKILL.md"),
		filepath.Join(home, ".gemini", "antigravity", "skills", "strategist", "SKILL.md"),
		filepath.Join(home, ".codex", "skills", "strategist", "SKILL.md"),
	}
	for _, p := range expectedPaths {
		data, err := os.ReadFile(p)
		require.NoError(t, err, "shim should exist at %s", p)
		assert.Contains(t, string(data), "# SKILL", "shim content at %s", p)
	}
}

func TestInstallOptionalShims_SkipsWhenDirAbsent(t *testing.T) {
	t.Parallel()
	home := t.TempDir() // no .gemini or .codex dirs

	installOptionalShims(home, "# SKILL", "")

	for _, dir := range []string{".gemini", ".codex"} {
		_, err := os.Stat(filepath.Join(home, dir))
		assert.True(t, os.IsNotExist(err), "optional dir %s should not be created", dir)
	}
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

	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "gitignore")
}

// --- stripFrontmatter ---

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()

	t.Run("no frontmatter returns string unchanged", func(t *testing.T) {
		t.Parallel()
		input := "# SKILL\nsome content\n"
		assert.Equal(t, input, stripFrontmatter(input))
	})

	t.Run("strips frontmatter block", func(t *testing.T) {
		t.Parallel()
		input := "---\nname: strategist\n---\n\n# SKILL\nbody\n"
		want := "# SKILL\nbody\n"
		assert.Equal(t, want, stripFrontmatter(input))
	})

	t.Run("unclosed frontmatter returns string unchanged", func(t *testing.T) {
		t.Parallel()
		input := "---\nname: strategist\nno closing marker\n"
		assert.Equal(t, input, stripFrontmatter(input))
	})
}

// --- loadKnownProviders ---

// knownProvidersExtractor serves a synthetic templates/known-providers.yaml.
type knownProvidersExtractor struct {
	yaml string
}

func (k knownProvidersExtractor) Extract(_ string, _ bool) error { return nil }
func (k knownProvidersExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == "templates/known-providers.yaml" {
		return []byte(k.yaml), nil
	}
	return nil, fmt.Errorf("not found: %s", relPath)
}

func TestLoadKnownProviders(t *testing.T) {
	t.Parallel()

	t.Run("reads valid providers yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers:\n  brainstorming: write_analysis\n  sdd-ask: controlled\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, "write_analysis", got["brainstorming"])
		assert.Equal(t, "controlled", got["sdd-ask"])
	})

	t.Run("falls back to static map when providers map is empty", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers: {}\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})

	t.Run("falls back to static map on invalid yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: ": invalid: yaml:\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})
}

// --- writeSelectedProviderManifests: ReadFile error path ---

// partialExtractor serves minimalExtractor files but fails for the given path.
type partialExtractor struct {
	failPath string
}

func (p partialExtractor) Extract(targetDir string, overwrite bool) error {
	return minimalExtractor{}.Extract(targetDir, overwrite)
}
func (p partialExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == p.failPath {
		return nil, fmt.Errorf("partialExtractor: injected failure for %s", relPath)
	}
	return minimalExtractor{}.ReadFile(relPath)
}

func TestWriteSelectedProviderManifests_ReadFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   partialExtractor{failPath: "skills/brainstorming/skill.yaml"},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
		WizardPrompter: NewTextPrompter(strings.NewReader(
			"en\nen\nen\nen\nepic\n.analysis\nyes\nbrainstorming\nopenspec-explore\nsdd-ask\n\n",
		)),
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "brainstorming")
}

// --- promptSlots: warning printed for unknown provider ---

func TestPromptSlots_UnknownProviderPrintsWarning(t *testing.T) {
	t.Parallel()
	// Use a custom provider that is NOT in knownProviderRisk → warning is emitted.
	input := "en\nen\nen\nen\nepic\n.analysis\nyes\ncustom-ranger\nopenspec-explore\nsdd-ask\n\n"
	wc, err := runWizard(NewTextPrompter(strings.NewReader(input)), minimalExtractor{})
	require.NoError(t, err)
	assert.Equal(t, "custom-ranger", wc.DiscoveryProvider)
}
