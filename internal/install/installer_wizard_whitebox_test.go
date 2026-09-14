package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
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

// TestInstall_WizardPath_PersistsPluginLock proves ADR-0037's DEC-001 end to
// end: a real `strategist install` (wizard mode) run against a scratch
// workspace produces plugins.lock with resolved bindings for both the
// discovery and refinement slots, rather than the resolution being computed
// and discarded on every invocation
// (.analysis/refined/20260913-wizard-plugin-lifecycle-persistence-gap/).
func TestInstall_WizardPath_PersistsPluginLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nbrainstorming\narchivist\nsdd-ask\n\n")
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))

	data, err := os.ReadFile(filepath.Join(dir, ".strategist", "plugins.lock"))
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "schema_version: strategist-plugin-lock-file/v1")
	assert.Contains(t, s, "slot: discovery")
	assert.Contains(t, s, "slot: refinement")
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
	// Neither brainstorming nor openspec-propose declares a
	// supported_handoff_schemas value matching its role's HandoffSchema
	// (honest — see their strategist.yaml sidecars and
	// .analysis/done/20260728-ranger-drift-eval/ for why), so
	// compatibleProviderOptions no longer offers them as the accept-defaults
	// choice — the wizard now defaults both slots to their native role.
	assert.Contains(t, s, "discovery: ranger")
	assert.Contains(t, s, "refinement: archivist")
	assert.Contains(t, s, "execution: sniper")

	// Neither slot's default is an installable skill package anymore
	// (both are native roles) — accepting defaults must not materialize any
	// skill.yaml manifest for either.
	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "openspec-propose", "skill.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
	// openspec-explore is not selected by defaults — it must not be materialized.
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
			_, err := runWizard(context.Background(), p(tt.input), minimalExtractor{}, "")
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
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nbrainstorming\nbrainstorming\narchivist\nsdd-ask\n\n")
	svc.AwarenessRefresher = func(strategistRoot, projectRoot, _ string) bool {
		called = true
		assert.Equal(t, filepath.Join(dir, ".strategist"), strategistRoot)
		assert.Equal(t, dir, projectRoot)
		return true
	}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))
	assert.True(t, called, "AwarenessRefresher must be called after wizard install")
}

// TestPromptSlots_UnknownProviderPrintsWarning exercises promptSlots directly
// rather than the full runWizard: validateProvider's "not in the known
// plugin catalog" warning is non-blocking at prompt time, but (correctly,
// per docs/adr/0029-external-skill-provider-lifecycle.md §4) an
// unresolved-to-the-catalog provider is unconditionally rejected later in
// runWizard by planPluginOnboarding's own catalog-membership check — the two
// are different, independent gates, and this test's scope is only the
// prompt-time warning, not full wizard completion (see
// TestRunWizardBlocksOnUnresolvedCustomSkill for the later, hard-blocking gate).
func TestPromptSlots_UnknownProviderPrintsWarning(t *testing.T) {
	t.Parallel()
	b := i18n.BundleFor("en")
	input := "custom-ranger\nopenspec-explore\nsdd-ask\n\n"
	catalog, err := parseCatalogBytes([]byte(minimalCatalogYAML))
	require.NoError(t, err)
	discovery, refinement, execution, err := promptSlots(NewTextPrompter(strings.NewReader(input)), b, catalog, knownProviderRisk)
	require.NoError(t, err)
	assert.Equal(t, "custom-ranger", discovery)
	assert.Equal(t, "openspec-explore", refinement)
	assert.Equal(t, nativeExecutionProvider, execution)
}

// TestRunWizardBlocksOnUnresolvedCustomSkill covers docs/adr/0029's converse
// case from TestPromptSlots_UnknownProviderPrintsWarning above: a slot
// provider that is neither a known registry/catalog entry nor resolvable as
// an already-installed workspace skill must hard-block the full wizard run
// (checkCustomSkillAvailability, the first of runWizard's two independent
// catalog-membership/local-resolution gates — see planPluginOnboarding for
// the second, stricter one), not just warn.
func TestRunWizardBlocksOnUnresolvedCustomSkill(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // no skill installed under homeDir — deliberately unresolvable

	input := "en\nen\nen\nen\nepic\n.analysis\ndefinitely-not-a-real-installed-skill-id-xyz\nopenspec-explore\nsdd-ask\n\n"
	_, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(input)), minimalExtractor{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configured_unverified")
}
