package install

import (
	"context"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
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
	assert.Contains(t, s, "execution: sdd-ask")
	assert.NotContains(t, s, "execution: sniper")

	brainstorming, err := os.ReadFile(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(brainstorming), "risk_score: write_analysis")
	assert.Contains(t, string(brainstorming), "invocation_evidence: required")
	assert.Contains(t, string(brainstorming), "native_substitution: forbidden")

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
	lockFile, err := readPluginLockFile(filepath.Join(dir, ".strategist"))
	require.NoError(t, err)
	assert.NotEmpty(t, lockFile.Lock.GraphDigest)
	assert.Len(t, lockFile.Lock.Nodes, 6)
	_, ok := findSlotBinding(lockFile.Bindings, "discovery")
	assert.True(t, ok)
	_, ok = findSlotBinding(lockFile.Bindings, "refinement")
	assert.True(t, ok)
	_, ok = findSlotBinding(lockFile.Bindings, "execution")
	assert.False(t, ok)
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
	// The wizard defaults to the embedded weapons affiliated with each role;
	// fixed role checkpoints still own handoff normalization.
	assert.Contains(t, s, "discovery: brainstorming")
	assert.Contains(t, s, "refinement: openspec-propose")
	assert.Contains(t, s, "execution: sniper")

	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "brainstorming", "skill.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, ".strategist", "skills", "openspec-propose", "skill.yaml"))
	require.NoError(t, err)
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

// TestInstall_WizardPath_PersistsRankedBindingModes protects the full wizard
// decision boundary: a Ranked selection must reach both active.yaml and the
// corresponding plugins.lock binding for each configurable role.
func TestInstall_WizardPath_PersistsRankedBindingModes(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	// No t.Parallel(): t.Setenv modifies the process-global PATH.
	dir := t.TempDir()
	// Inject a minimal fake openspec binary so prepareRankedProviderRuntimes
	// can bootstrap the openspec-propose runtime without requiring the real
	// openspec executable in $PATH (which is absent on CI runners).
	binDir := t.TempDir()
	openspecBin := filepath.Join(binDir, "openspec")
	fakeScript := `#!/bin/sh
set -eu
if [ "$1" = "init" ]; then
  mkdir -p "$PWD/openspec"
  printf 'schema: spec-driven\n' > "$PWD/openspec/config.yaml"
  exit 0
fi
printf '{"root":{"path":"%s"},"members":[],"status":[]}\n' "$(dirname "$PWD")"
`
	require.NoError(t, os.WriteFile(openspecBin, []byte(fakeScript), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	input := "en\nen\nen\nen\nepic\n.analysis\nbrainstorming::ranked\nopenspec-propose::ranked\nsniper::ranked\n\n"
	svc := Service{Extractor: rankedWizardExtractor{}, Compiler: nopCompiler{}, WizardPrompter: NewTextPrompter(strings.NewReader(input)), ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))

	active, err := os.ReadFile(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(active), "discovery: brainstorming")
	assert.Contains(t, string(active), "refinement: openspec-propose")
	assert.Contains(t, string(active), "execution: sniper")

	lock, err := readPluginLockFile(filepath.Join(dir, ".strategist"))
	require.NoError(t, err)
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		binding, ok := findSlotBinding(lock.Bindings, slot)
		require.True(t, ok, "wizard must persist a binding for %s", slot)
		assert.Equal(t, domain.SlotBindingModeRanked, binding.Mode, "wizard Ranked choice must survive persistence for %s", slot)
	}
}

func TestInstall_WizardRankedPathBootstrapsContainedOpenSpecRuntime(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	dir := t.TempDir()
	t.Setenv("OPEN_SPEC_CONFIG", filepath.Join(t.TempDir(), "foreign-config.yaml"))

	input := "en\nen\nen\nen\nepic\n.analysis\nbrainstorming::ranked\nopenspec-propose::ranked\nsniper::ranked\n\n"
	svc := Service{Extractor: rankedWizardExtractor{}, Compiler: nopCompiler{}, WizardPrompter: NewTextPrompter(strings.NewReader(input)), ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))

	strategist := filepath.Join(dir, ".strategist")
	config := filepath.Join(strategist, "openspec", "config.yaml")
	require.FileExists(t, config)
	assert.NoDirExists(t, filepath.Join(strategist, "openspec", "openspec"))
	assert.NoDirExists(t, filepath.Join(dir, "openspec"))
	assert.FileExists(t, filepath.Join(strategist, "ranked-runtimes.yaml"))

	state, err := os.ReadFile(filepath.Join(strategist, "ranked-runtimes.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(state), "openspec-propose")
	assert.Contains(t, string(state), "sha256:openspec-propose")
	assert.NotContains(t, string(state), `"mode":`)
	assert.Contains(t, string(state), `"node": "/`)
	assert.Contains(t, string(state), `"script": "weapon-runtime/openspec-propose/openspec/`)
}

type rankedWizardExtractor struct{}

func (rankedWizardExtractor) Extract(targetDir string, withShim bool) error {
	if err := (minimalExtractor{}).Extract(targetDir, withShim); err != nil {
		return err
	}
	if err := copyOpenSpecRuntimeFixture(targetDir); err != nil {
		return err
	}
	roleFiles := map[string]string{
		"roles/default.yaml":   "discovery: ranger\nrefinement: archivist\nexecution: sniper\n",
		"roles/ranger.yaml":    "role: ranger\nslot: discovery\n",
		"roles/archivist.yaml": "role: archivist\nslot: refinement\n",
		"roles/sniper.yaml":    "role: sniper\nslot: execution\n",
	}
	for path, content := range roleFiles {
		if err := os.WriteFile(filepath.Join(targetDir, path), []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "plugins"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(targetDir, pluginCatalogPath), []byte(rankedWizardCatalogYAML), 0o644); err != nil {
		return err
	}
	return nil
}

func (rankedWizardExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == pluginCatalogPath {
		return []byte(rankedWizardCatalogYAML), nil
	}
	return minimalExtractor{}.ReadFile(relPath)
}

const rankedWizardCatalogYAML = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: archivist
    risk_score: write_analysis
    compatibility_source: native_role
  - id: brainstorming
    risk_score: write_analysis
    canonical_role: ranger
    default: true
    ranked: true
    certification_digest: sha256:brainstorming
    ranked_binding_generation: 1
    ranked_binding_status: active
    installable: true
    legacy_manifest_path: skills/brainstorming/skill.yaml
    compatibility_source: embedded
  - id: openspec-propose
    risk_score: write_analysis
    canonical_role: archivist
    default: true
    ranked: true
    certification_digest: sha256:openspec-propose
    ranked_binding_generation: 1
    ranked_binding_status: active
    runtime:
      kind: openspec_root
      root: .strategist/openspec
      bootstrap: openspec init --profile core --tools codex
      healthcheck: openspec context --json
    installable: true
    legacy_manifest_path: skills/openspec-propose/skill.yaml
    compatibility_source: embedded
  - id: sniper
    risk_score: controlled
    canonical_role: sniper
    ranked: true
    certification_digest: sha256:sniper
    compatibility_source: native_role
`

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
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\nbrainstorming\nopenspec-propose\narchivist\nsdd-ask\n\n")
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
	discovery, refinement, execution, _, _, _, err := promptSlots(NewTextPrompter(strings.NewReader(input)), b, catalog, knownProviderRisk)
	require.NoError(t, err)
	assert.Equal(t, "custom-ranger", discovery)
	assert.Equal(t, "openspec-explore", refinement)
	assert.Equal(t, "sdd-ask", execution)
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

func TestRunWizardBlocksOnUnresolvedCustomExecutionProvider(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	input := "en\nen\nen\nen\nepic\n.analysis\nbrainstorming\nopenspec-propose\ncustom-execution\n\n"
	_, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(input)), minimalExtractor{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configured_unverified")
}
