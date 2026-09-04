package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

func TestInstall_WizardPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nbrainstorming\narchivist\nsdd-ask\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, readErr)
	s := string(data)
	assert.Contains(t, s, "mode: epic")
	assert.Contains(t, s, "base_path: /workspace")
	assert.NotContains(t, s, "roles_config")
	assert.Contains(t, s, "ui: en")
	assert.Contains(t, s, "docs: en")
	assert.Contains(t, s, "chat: pt-BR")
	assert.Contains(t, s, "code: en")
	assert.NotContains(t, s, "adr_enabled")
	assert.NotContains(t, s, "execution_mode")
	assert.NotContains(t, s, "git_persistence_mode")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sniper")
	assert.NotContains(t, s, "execution: sdd-ask")

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestInstall_WizardPath_WithChest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\nen\nen\npragmatic\n.analysis\nbrainstorming\nopenspec-explore\nsdd-ask\n.sdd/source\n")
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
	svc := newSvcW(t, "\n\n\n\n\n\n\n\n\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	s := string(data)
	assert.Contains(t, s, "mode: epic")
	assert.NotContains(t, s, "roles_config")
	assert.Contains(t, s, "ui: en")
	assert.Contains(t, s, "docs: en")
	assert.Contains(t, s, "chat: en")
	assert.Contains(t, s, "code: en")
	assert.NotContains(t, s, "adr_enabled")
	assert.NotContains(t, s, "execution_mode")
	assert.NotContains(t, s, "git_persistence_mode")
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sniper")

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "id: brainstorming")
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	// archivist is the native refinement role, not an installable skill package —
	// accepting defaults must not require a separately installed openspec-explore.
	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestInstall_WizardPath_ExplicitDefaultProvidersMaterializeManifests(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\nen\nen\nepic\n.analysis\nbrainstorming\nopenspec-explore\nsdd-ask\n\n")
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.NoError(t, err)

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")

	openspecExplore, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "openspec-explore", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(openspecExplore), "risk_score: write_analysis")
}

func TestRunWizard_EOFPrompts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{name: "first", input: "", wantError: "ui_language"},
		{name: "second", input: "en\n", wantError: "doc_language"},
		{name: "third chat", input: "en\nen\n", wantError: "chat_language"},
		{name: "fourth code", input: "en\nen\nen\n", wantError: "code_language"},
		{name: "fifth mode", input: "en\nen\nen\nen\n", wantError: "mode"},
		{name: "sixth base path", input: "en\nen\nen\nen\npragmatic\n", wantError: "base_path"},
		{name: "seventh discovery", input: "en\nen\nen\nen\npragmatic\n.\n", wantError: "discovery"},
		{name: "eighth refinement", input: "en\nen\nen\nen\npragmatic\n.\nbrainstorming\n", wantError: "refinement"},
		{name: "ninth execution", input: "en\nen\nen\nen\npragmatic\n.\nbrainstorming\nopenspec-explore\n", wantError: "execution"},
		{name: "tenth chest", input: "en\nen\nen\nen\npragmatic\n.\nbrainstorming\nopenspec-explore\nsdd-ask\n", wantError: "treasure_chest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := runWizard(context.Background(), p(tt.input), minimalExtractor{})
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestApplyConfig_NilPrompter_NonTTY_FailsOnEmptyStdin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:        minimalExtractor{},
		Compiler:         nopCompiler{},
		ShimHomeDir:      t.TempDir(),
		terminalDetector: func() bool { return false },
		stdinReader:      strings.NewReader(""),
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
		terminalDetector: func() bool { return true },
		tuiPrompterFn:    func() Prompter { return &TUIPrompter{runFn: errRun} },
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
		stdinReader: strings.NewReader(""),
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "wizard")
}

func TestInstall_WizardPath_AwarenessRefresherCalled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	called := false
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nyes\nbrainstorming\narchivist\nsdd-ask\n\n")
	svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
		called = true
		assert.Equal(t, filepath.Join(dir, ".strategist"), strategistRoot)
		assert.Equal(t, dir, projectRoot)
		return true
	}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))
	assert.True(t, called, "AwarenessRefresher must be called after wizard install")
}

func TestPromptSlots_UnknownProviderPrintsWarning(t *testing.T) {
	t.Parallel()
	input := "en\nen\nen\nen\nepic\n.analysis\ncustom-ranger\nopenspec-explore\nsdd-ask\n\n"
	wc, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(input)), minimalExtractor{})
	require.NoError(t, err)
	assert.Equal(t, "custom-ranger", wc.DiscoveryProvider)
}
