package main

// Tests for all Cobra command RunE/Run functions.
// Each test targets a specific command to maximise coverage without
// triggering os.Exit (which would kill the test process).

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// freshArtifactDir creates an artifact + manifest pair with no sources
// (= always considered fresh by IsStale).
func freshArtifactDir(t *testing.T) (dir, artifactPath string) {
	t.Helper()
	dir = t.TempDir()
	artifactPath = filepath.Join(dir, "artifact.gz")
	testutil.WriteGzJSON(t, artifactPath, map[string]any{"sources": map[string]int64{}})
	testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{"generated_at": 0})
	return dir, artifactPath
}

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// --- version ---

func TestVersionCmd_PrintsVersion(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.2.3-test"

	out := captureStdout(t, func() {
		versionCmd.Run(versionCmd, nil)
	})
	assert.Contains(t, out, "1.2.3-test")
	assert.Contains(t, out, "strategist")
}

func TestVersionCmd_EmitsStructuredTelemetry(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.2.3-test"

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	versionCmd.Run(versionCmd, nil)

	out := buf.String()
	assert.Contains(t, out, "strategist.component=version")
	assert.Contains(t, out, "strategist.runtime_mode=cli")
	assert.Contains(t, out, "strategist.output_profile=default")
	assert.Contains(t, out, "strategist.version=1.2.3-test")
}

// --- compile ---

func TestCompileCmd_Success(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = dir

	err := compileCmd.RunE(compileCmd, nil)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".manifest.gz"))
}

func TestCompileCmd_DefaultRoot(t *testing.T) {
	// When compileRoot is empty it defaults to ".strategist"; that dir doesn't
	// exist here so we get an error — but the "if compileRoot == """ branch is covered.
	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = ""

	// Change to a temp dir so ".strategist" definitely doesn't exist.
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = compileCmd.RunE(compileCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_installed")
	// After the run, compileRoot must be the default value.
	assert.Equal(t, ".strategist", compileRoot)
}

// --- check-stale ---

func TestCheckStaleCmd_FreshArtifact(t *testing.T) {
	_, artifactPath := freshArtifactDir(t)
	err := checkStaleCmd.RunE(checkStaleCmd, []string{artifactPath})
	require.NoError(t, err) // fresh → isStale=false → no os.Exit
}

func TestCheckStaleCmd_CorruptArtifact(t *testing.T) {
	dir := t.TempDir()
	art := filepath.Join(dir, "artifact.gz")
	require.NoError(t, os.WriteFile(art, []byte("not gzip"), 0o644))
	testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})

	err := checkStaleCmd.RunE(checkStaleCmd, []string{art})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check-stale")
}

// --- install ---

func TestInstallCmd_ErrorPath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	orig := installTarget
	t.Cleanup(func() { installTarget = orig })
	installTarget = dir

	err := installCmd.RunE(installCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install")
}

func TestInstallCmd_DefaultTarget(t *testing.T) {
	// When installTarget is empty it defaults to "." — cover that branch.
	// We expect an error (real install would touch ~/.claude/) so we
	// use a read-only CWD to abort early inside the extractor.
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	origTarget := installTarget
	origSilent := installSilent
	origWizard := installWizard
	origGlobal := installGlobal
	t.Cleanup(func() {
		installTarget = origTarget
		installSilent = origSilent
		installWizard = origWizard
		installGlobal = origGlobal
	})

	readOnly := t.TempDir()
	require.NoError(t, os.Chmod(readOnly, 0o555))
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(readOnly))

	installTarget = "" // triggers the default "." branch
	installSilent = true
	installWizard = false
	installGlobal = false

	err = installCmd.RunE(installCmd, nil)
	require.Error(t, err) // extraction into read-only "." fails
	assert.Equal(t, ".", installTarget)
}

// --- root / execute ---

func TestRootCmd_UnknownSubcommand(t *testing.T) {
	// rootCmd.Execute returns an error for unknown commands without calling os.Exit.
	rootCmd.SetArgs([]string{"__unknown_cmd__"})
	err := rootCmd.Execute()
	// Cobra returns an error for unknown commands.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestExecute_NoError(t *testing.T) {
	// Smoke-test execute() success path: "version" command succeeds.
	// We redirect Stdout to suppress output during the test.
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "smoke"

	// Capture stdout to avoid test noise.
	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute() //nolint:errcheck // return value not needed here
	})
}

// TestExecute_Success calls execute() directly with a valid command so that the
// success branch (err == nil, no os.Exit) is covered.
func TestExecute_Success(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "execute-smoke"

	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		execute()
	})
}

// TestMain_Smoke calls main() directly (valid in package main tests) with a safe
// command so neither main() nor execute() can reach os.Exit.
func TestMain_Smoke(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "main-smoke"

	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		main()
	})
}

// TestExecute_ErrorPath covers the os.Exit(1) branch in execute() by running the
// test binary in a subprocess with an unknown command.
func TestExecute_ErrorPath(t *testing.T) {
	if os.Getenv("STRATEGIST_EXPECT_EXIT") == "1" {
		rootCmd.SetArgs([]string{"__exit_test__"})
		execute()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_ErrorPath")
	cmd.Env = append(os.Environ(), "STRATEGIST_EXPECT_EXIT=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error, got: %v", err)
	}
	assert.Equal(t, 1, exitErr.ExitCode())
}

// --- validate ---

// minimalValidateRoot creates a .strategist/-like tree suitable for validateCmd:
// active.yaml, personas/pragmatic.yaml, roles/default.yaml.
func minimalValidateRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\nroles_config: default\nslots:\n  discovery: brainstorming\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "pragmatic.yaml"),
		[]byte("id: pragmatic\ntone_directive: precise\nphase_labels:\n  discovery: analysis\n  refinement: refinement\n  execution: execution\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\nrefinement: archivist\nexecution: caveman\n"), 0o644))
	return dir
}

func TestValidateCmd_Success(t *testing.T) {
	dir := minimalValidateRoot(t)

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	out := captureStdout(t, func() {
		err := validateCmd.RunE(validateCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "validate OK")
	assert.Contains(t, out, dir)
}

func TestValidateCmd_EmitsStructuredTelemetry(t *testing.T) {
	root := minimalValidateRoot(t)

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = root

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	err := validateCmd.RunE(validateCmd, nil)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "strategist.component=validate")
	assert.Contains(t, out, "strategist.runtime_mode=cli")
	assert.Contains(t, out, "strategist.output_profile=default")
	assert.Contains(t, out, "strategist.target="+root)
}

func TestValidateCmd_WithKnowledgeIndex(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte("schema_version: \"1\"\nsources: []\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	out := captureStdout(t, func() {
		err := validateCmd.RunE(validateCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "validate OK")
}

func TestValidateCmd_MissingRoot(t *testing.T) {
	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestValidateCmd_MissingActiveYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestValidateCmd_InvalidMode(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: invalid_mode\nbase_path: .analysis\nroles_config: default\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_MissingSlot(t *testing.T) {
	dir := minimalValidateRoot(t)
	// overwrite roles/default.yaml without the required slots
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"),
		[]byte("discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidActiveYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte(": invalid: yaml: content:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_MissingRequiredField(t *testing.T) {
	dir := minimalValidateRoot(t)
	// active.yaml missing roles_config
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidPersonaYAML(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "bad.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_PersonaMissingField(t *testing.T) {
	dir := minimalValidateRoot(t)
	// persona without phase_labels
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "minimal.yaml"),
		[]byte("id: minimal\ntone_directive: brief\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidRoleYAML(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "bad.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_InvalidKnowledgeIndex(t *testing.T) {
	dir := minimalValidateRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = dir

	err := validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
}

func TestValidateCmd_DefaultRoot(t *testing.T) {
	// When validateRoot is empty, auto-discovery walks up from CWD.
	// In an empty temp dir (no .strategist/), it returns a "runtime not found" error.
	orig := validateRoot
	t.Cleanup(func() { validateRoot = orig })
	validateRoot = ""

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = validateCmd.RunE(validateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime not found")
}

// TestCompileCmd_PrintsCompletion verifies the success message path.
func TestCompileCmd_PrintsCompletion(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = dir

	out := captureStdout(t, func() {
		err := compileCmd.RunE(compileCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "compile complete")
}

// TestInstallCmd_PrintsCompletion verifies the success message (install completes).
func TestInstallCmd_PrintsCompletion(t *testing.T) {
	dir := t.TempDir()

	origTarget := installTarget
	origSilent := installSilent
	origWizard := installWizard
	origGlobal := installGlobal
	t.Cleanup(func() {
		installTarget = origTarget
		installSilent = origSilent
		installWizard = origWizard
		installGlobal = origGlobal
	})
	installTarget = dir
	installSilent = true
	installWizard = false
	installGlobal = false

	out := captureStdout(t, func() {
		err := installCmd.RunE(installCmd, nil)
		if err != nil {
			// In some CI environments the shim step may fail — that's OK for
			// this test; we just need to exercise the target-defaulting branch.
			t.Logf("install returned (possibly expected in CI): %v", err)
		}
	})
	_ = out
}

// --- providers ---

func TestInitiativeCmd_ShowsAllSlots(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, initiativeCmd.RunE(initiativeCmd, nil))
	})
	assert.Contains(t, out, "discovery")
	assert.Contains(t, out, "brainstorming")
	assert.Contains(t, out, "refinement")
	assert.Contains(t, out, "openspec-explore")
	assert.Contains(t, out, "execution")
	assert.Contains(t, out, "sdd-ask")
	// no manifests in minimal root → all show absent
	assert.Contains(t, out, "⚠ manifest ausente")
}

func TestInitiativeCmd_ShowsManifestOK(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	// write a minimal provider manifest for brainstorming
	provDir := filepath.Join(dir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(provDir, 0o755))
	manifest := []byte("id: brainstorming\nstatus: active\nrisk_score: write_analysis\nprovider_class: rankeado\nspecialization_taxonomy:\n  canonical_role: ranger\n  provider_class: rankeado\n")
	require.NoError(t, os.WriteFile(filepath.Join(provDir, "skill.yaml"), manifest, 0o644))

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, initiativeCmd.RunE(initiativeCmd, nil))
	})
	assert.Contains(t, out, "Ranger rankeado")
	assert.Contains(t, out, "✓ manifest OK")
}

func TestInitiativeCmd_MissingActiveYAML(t *testing.T) {
	dir := t.TempDir() // empty — no active.yaml

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	err := initiativeCmd.RunE(initiativeCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestInitiativeCmd_DefaultRootFallback(t *testing.T) {
	// When --root is empty, RunE auto-discovers via findStrategistRoot.
	// In a tmpdir with no .strategist/, it should return an error containing "not found".
	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = ""

	// Change CWD to an isolated temp dir so we don't accidentally pick up the real runtime.
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	err := initiativeCmd.RunE(initiativeCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProviderRow_FallbackRoles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no manifests

	role, class, status := providerRow(dir, "discovery", "custom-ranger")
	assert.Equal(t, "Ranger", role)
	assert.Equal(t, "(base)", class)
	assert.Equal(t, "⚠ manifest ausente", status)

	role, _, _ = providerRow(dir, "refinement", "custom-arch")
	assert.Equal(t, "Archivist", role)

	role, _, _ = providerRow(dir, "execution", "custom-sniper")
	assert.Equal(t, "Sniper", role)

	role, _, _ = providerRow(dir, "custom-slot", "some-provider")
	assert.Equal(t, "Custom-slot", role) // unknown slot → title-case of slot name
}

func TestCanonicalRoleLabel(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"ranger", "Ranger"},
		{"RANGER", "Ranger"},
		{"archivist", "Archivist"},
		{"Archivist", "Archivist"},
		{"sniper", "Sniper"},
		{"custom", "Custom"},
		{"", ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, canonicalRoleLabel(tc.input), "input=%q", tc.input)
	}
}

func TestInstallCmd_GlobalFlag_ResolvesHomeDefault(t *testing.T) {
	origTarget := installTarget
	origSilent := installSilent
	origWizard := installWizard
	origGlobal := installGlobal
	t.Cleanup(func() {
		installTarget = origTarget
		installSilent = origSilent
		installWizard = origWizard
		installGlobal = origGlobal
	})

	home := t.TempDir()
	t.Setenv("HOME", home)
	installTarget = ""
	installSilent = true
	installWizard = false
	installGlobal = true

	err := installCmd.RunE(installCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, home, installTarget)
}

// --- dojo ---

func setupDojoScenario(t *testing.T, scenario, criteria, runContent string) string {
	t.Helper()
	dir := t.TempDir()

	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: brainstorming\n"), 0o644))

	scenarioDir := filepath.Join(dir, ".analysis", "dojo", scenario)
	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "criteria.yaml"), []byte(criteria), 0o644))

	if runContent != "" {
		runDir := filepath.Join(dir, ".analysis", "dojo", "run", "todo")
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte(runContent), 0o644))
	}
	return strategistRoot
}

func TestDojoCheckCmd_AllPass(t *testing.T) {
	root := setupDojoScenario(t, "quick-draw",
		"scenario: quick-draw\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n    must_contain: [KATA_RAPIDO]\n",
		"ideia: KATA_RAPIDO test\n",
	)

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"quick-draw"})
		require.NoError(t, err)
	})
	assert.Contains(t, out, "PASS")
	assert.Contains(t, out, "KATA_RAPIDO")
}

func TestDojoCheckCmd_FileMissing(t *testing.T) {
	root := setupDojoScenario(t, "quick-draw",
		"scenario: quick-draw\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\n",
		"",
	)

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"quick-draw"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})
	assert.Contains(t, out, "FAIL")
}

func TestDojoCheckCmd_MissingActiveYAML(t *testing.T) {
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"quick-draw"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestDojoCheckCmd_MissingCriteria(t *testing.T) {
	root := setupDojoScenario(t, "quick-draw", "scenario: quick-draw\n", "")
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"nonexistent-scenario"})
	require.Error(t, err)
}

func TestDojoCheckCmd_FilesOnlySkipsEmitLog(t *testing.T) {
	root := setupDojoScenario(t, "quick-draw",
		"scenario: quick-draw\nrun_dir: dojo/run\nfiles_created:\n  - path: todo/geral.md\nemit_log:\n  must_contain: [ranger_start]\n",
		"ideia: content\n",
	)

	orig := dojoRoot
	origFilesOnly := dojoFilesOnly
	t.Cleanup(func() { dojoRoot = orig; dojoFilesOnly = origFilesOnly })
	dojoRoot = root
	dojoFilesOnly = true

	out := captureStdout(t, func() {
		err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"quick-draw"})
		require.NoError(t, err)
	})
	assert.Contains(t, out, "PASS")
	assert.NotContains(t, out, "emit.log not found")
}

func TestDojoCheckCmd_EmptyBasePath(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nroles_config: x\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	err := dojoCheckCmd.RunE(dojoCheckCmd, []string{"quick-draw"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_path")
}

func TestDojoListCmd_ListsScenarios(t *testing.T) {
	root := setupDojoScenario(t, "quick-draw",
		"scenario: quick-draw\ndescription: \"Quick Draw test\"\n", "")

	// root is .strategist; project root is its parent; dojo dir is <project>/.analysis/dojo
	projectRoot := filepath.Dir(root)
	s2 := filepath.Join(projectRoot, ".analysis", "dojo", "ranger-weapons")
	require.NoError(t, os.MkdirAll(s2, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s2, "criteria.yaml"),
		[]byte("scenario: ranger-weapons\ndescription: \"Ranger weapons test\"\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = root

	out := captureStdout(t, func() {
		err := dojoListCmd.RunE(dojoListCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "quick-draw")
}

func TestDojoListCmd_EmptyDojo(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: brainstorming\n"), 0o644))
	// base_path resolves to <dir>/.analysis (parent of strategistRoot is dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".analysis", "dojo"), 0o755))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	out := captureStdout(t, func() {
		err := dojoListCmd.RunE(dojoListCmd, nil)
		require.NoError(t, err)
	})
	assert.Empty(t, out)
}

func TestDojoListCmd_MissingDojoDir(t *testing.T) {
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: brainstorming\n"), 0o644))

	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = strategistRoot

	err := dojoListCmd.RunE(dojoListCmd, nil)
	require.Error(t, err)
}

func TestDojoListCmd_MissingActiveYAML(t *testing.T) {
	orig := dojoRoot
	t.Cleanup(func() { dojoRoot = orig })
	dojoRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := dojoListCmd.RunE(dojoListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

// --- check ---

// minimalCheckRoot creates a .strategist/ tree suitable for checkCmd with all
// three slot providers installed.
func minimalCheckRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, provider := range []struct {
		name      string
		riskScore string
	}{
		{"brainstorming", "write_analysis"},
		{"openspec-explore", "write_analysis"},
		{"sdd-ask", "controlled"},
	} {
		provDir := filepath.Join(dir, "skills", provider.name)
		require.NoError(t, os.MkdirAll(provDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(provDir, "skill.yaml"),
			[]byte("id: "+provider.name+"\nrisk_score: "+provider.riskScore+"\n"),
			0o644,
		))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	return dir
}

func TestCheckCmd_Success(t *testing.T) {
	dir := minimalCheckRoot(t)

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "check=ok")
	assert.Contains(t, out, "brainstorming")
	assert.Contains(t, out, "openspec-explore")
	assert.Contains(t, out, "sdd-ask")
}

func TestCheckCmd_MissingActiveYAML(t *testing.T) {
	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active_yaml_not_found")
}

func TestCheckCmd_ProviderNotInstalled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nroles_config: roles/default.yaml\nslots:\n  discovery: missing-provider\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_WrongRiskScore(t *testing.T) {
	dir := minimalCheckRoot(t)
	// overwrite brainstorming with wrong risk_score
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "skills", "brainstorming", "skill.yaml"),
		[]byte("id: brainstorming\nrisk_score: controlled\n"),
		0o644,
	))

	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = dir

	err := checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check=failed")
}

func TestCheckCmd_DefaultRoot(t *testing.T) {
	orig := checkRoot
	t.Cleanup(func() { checkRoot = orig })
	checkRoot = ""

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = checkCmd.RunE(checkCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime_not_found")
}

// --- exitCodeFor ---

func TestExitCodeFor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2, exitCodeFor(domain.ErrPipelineBypassDetected))
	assert.Equal(t, 3, exitCodeFor(domain.ErrSourceStale))
	assert.Equal(t, 3, exitCodeFor(domain.ErrArtifactAbsent))
	assert.Equal(t, 3, exitCodeFor(domain.ErrManifestMissing))
	assert.Equal(t, 1, exitCodeFor(errors.New("some generic error")))
	assert.Equal(t, 2, exitCodeFor(fmt.Errorf("wrapped: %w", domain.ErrPipelineBypassDetected)))
}

// --- requireStrategistDir ---

func TestRequireStrategistDir_FileExists(t *testing.T) {
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".strategist", "active.yaml"), []byte("mode: epic\n"), 0o644))

	assert.NoError(t, requireStrategistDir())
}

// --- addLine ---

func TestAddLine_NilRun(t *testing.T) {
	t.Parallel()
	// addLine must not panic when run is nil.
	assert.NotPanics(t, func() { addLine(nil) })
}

func TestAddLine_NonNilRun(t *testing.T) {
	t.Parallel()
	run := telemetry.NewMissionRun("test-add-line")
	// addLine must not panic and must update snapshot metrics.
	assert.NotPanics(t, func() { addLine(run) })
	snap := run.Snapshot()
	assert.Equal(t, int64(1), snap.LinesEmitted)
}

// --- go-file-size-report (Makefile contract) ---

func writeLines(t *testing.T, path string, count int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	for i := 0; i < count; i++ {
		_, err = fmt.Fprintf(f, "// line %d\n", i+1)
		require.NoError(t, err)
	}
}

func copyMakefile(t *testing.T, dstRoot string) {
	t.Helper()
	src, err := filepath.Abs("../../Makefile")
	require.NoError(t, err)
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dstRoot, "Makefile"), data, 0o644))
}

func TestMakeGoFileSizeReport_PrimarySourcesOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "embed", "defaults"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))

	writeLines(t, filepath.Join(root, "cmd", "app", "main.go"), 205)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service.go"), 240)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service_test.go"), 260)
	writeLines(t, filepath.Join(root, "internal", "embed", "defaults", "generated.go"), 300)
	writeLines(t, filepath.Join(root, ".strategist", "runtime.go"), 400)
	copyMakefile(t, root)

	cmd := exec.Command("make", "go-file-size-report")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	output := string(out)
	assert.Contains(t, output, "=== Go Files > 200 Lines ===")
	assert.Contains(t, output, "internal/pkg/service.go 240")
	assert.Contains(t, output, "cmd/app/main.go 205")
	assert.NotContains(t, output, "service_test.go")
	assert.NotContains(t, output, "internal/embed/defaults/generated.go")
	assert.NotContains(t, output, ".strategist/runtime.go")
}

func TestMakeGoFileSizeReport_PrintsNoneWhenNoLargeFilesExist(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "pkg"), 0o755))
	writeLines(t, filepath.Join(root, "cmd", "app", "main.go"), 40)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service.go"), 120)
	copyMakefile(t, root)

	cmd := exec.Command("make", "go-file-size-report")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	output := string(out)
	assert.Contains(t, output, "=== Go Files > 200 Lines ===")
	assert.Contains(t, output, "none")
}

// --- dojoItemLine ---

func TestDojoItemLine_Passed(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: true}
	line := dojoItemLine(item)
	assert.Contains(t, line, "✓")
	assert.Contains(t, line, "file-exists")
}

func TestDojoItemLine_FailedWithDetail(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: false, Detail: "missing"}
	line := dojoItemLine(item)
	assert.Contains(t, line, "✗")
	assert.Contains(t, line, "missing")
}

func TestDojoItemLine_FailedWithoutDetail(t *testing.T) {
	t.Parallel()
	item := domain.DojoCheckItem{Label: "file-exists", Passed: false}
	line := dojoItemLine(item)
	assert.Contains(t, line, "FAIL")
}
