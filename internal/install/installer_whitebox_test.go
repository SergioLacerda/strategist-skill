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
		filepath.Join(targetDir, "index.yaml"):                             "load_always: []\nload_by_task_type: {}\n",
		filepath.Join(targetDir, "templates", "pragmatic-standalone.yaml"): "mode: pragmatic\nbase_path: .analysis\n",
		filepath.Join(targetDir, "templates", "epic-standalone.yaml"):      "mode: epic\nbase_path: .analysis\n",
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
		return []byte("mode: epic\nbase_path: .analysis\n"), nil
	case "SKILL.md":
		return []byte("# SKILL\n"), nil
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
	// 12 prompts: ui/doc/chat/code/mode/base/adr/missionMode/discovery/refinement/execution/chest
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nyes\nentrega_executada\nbrainstorming\narchivist\nsdd-ask\n\n")
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
	assert.Contains(t, s, "mission_mode: entrega_executada")
	assert.Contains(t, s, "escopo_done: entrega")
	assert.Contains(t, s, "aplicar_alteracoes: true")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sdd-ask")
}

func TestInstall_WizardPath_WithChest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// 12 prompts
	svc := newSvcW(t, "en\nen\nen\nen\npragmatic\n.analysis\nyes\nanalise\nbrainstorming\nopenspec-explore\nsdd-ask\n.sdd/source\n")
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
	svc := newSvcW(t, "\n\n\n\n\n\n\n\n\n\n\n\n") // all defaults (12 prompts)
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
	assert.Contains(t, s, "mission_mode: entrega_executada")
	assert.Contains(t, s, "escopo_done: entrega")
	assert.Contains(t, s, "aplicar_alteracoes: true")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: openspec-explore")
	assert.Contains(t, s, "execution: sdd-ask")
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
	assert.Equal(t, "mode: epic\nbase_path: .analysis\n", string(got), "--force must overwrite active.yaml with embedded template")
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
		Mode: "pragmatic", BasePath: ".", MissionMode: "analise", DoneScope: "analise", ApplyChanges: false, UILanguage: "pt", DocLanguage: "pt", ChatLanguage: "pt", CodeLanguage: "pt", AdrEnabled: true,
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-explore", ExecutionProvider: "sdd-ask",
	})
	require.Error(t, err)
}

func p(input string) Prompter { return NewTextPrompter(strings.NewReader(input)) }

func TestRunWizard_EOFOnFirstPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p(""))
	require.Error(t, err)
	assert.ErrorContains(t, err, "ui_language")
}

func TestRunWizard_EOFOnSecondPrompt(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "doc_language")
}

func TestRunWizard_EOFOnThirdPrompt_ChatLang(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "chat_language")
}

func TestRunWizard_EOFOnFourthPrompt_CodeLang(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "code_language")
}

func TestRunWizard_EOFOnFifthPrompt_Mode(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "mode")
}

func TestRunWizard_EOFOnSixthPrompt_BasePath(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "base_path")
}

func TestRunWizard_EOFOnSeventhPrompt_Adr(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "adr_enabled")
}

func TestRunWizard_EOFOnEighthPrompt_Discovery(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "mission_mode")
}

func TestRunWizard_EOFOnNinthPrompt_DiscoveryAfterMissionMode(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nanalise\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "discovery")
}

func TestRunWizard_EOFOnTenthPrompt_Refinement(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nanalise\nbrainstorming\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "refinement")
}

func TestRunWizard_EOFOnEleventhPrompt_Execution(t *testing.T) {
	t.Parallel()
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nanalise\nbrainstorming\nopenspec-explore\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "execution")
}

func TestRunWizard_EOFOnTwelfthPrompt_ChestPath(t *testing.T) {
	t.Parallel()
	// All prompts answered except chest path (12th)
	_, err := runWizard(p("en\nen\nen\nen\npragmatic\n.\nyes\nanalise\nbrainstorming\nopenspec-explore\nsdd-ask\n"))
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
