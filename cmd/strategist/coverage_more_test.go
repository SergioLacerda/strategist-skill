package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimulateReport_ErrorAndWriterFailure(t *testing.T) {
	okOut := captureStdout(t, func() {
		require.NoError(t, printSimulateReport("/tmp/root", map[string]string{
			"discovery":  "ranger",
			"refinement": "archivist",
			"execution":  "sniper",
		}, map[string]slotResolution{
			"discovery":  {kind: slotResolutionSkillProvider},
			"refinement": {kind: slotResolutionSkillProvider},
			"execution":  {kind: slotResolutionNativeRole},
		}, "epic", "test", nil))
	})
	assert.Contains(t, okOut, "status=ready")
	assert.Contains(t, okOut, "kind=native_role")

	out := captureStdout(t, func() {
		err := printSimulateReport("/tmp/root", map[string]string{
			"discovery": "ranger",
			"execution": "sniper",
		}, map[string]slotResolution{
			"discovery": {kind: slotResolutionSkillProvider},
			"execution": {kind: slotResolutionNativeRole},
		}, "epic", "test", []string{"missing refinement"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "errors=1")
	})
	assert.Contains(t, out, "missing_provider")
	assert.Contains(t, out, "missing refinement")

	sw := &simReportWriter{w: tabwriter.NewWriter(errorWriter{}, 0, 0, 3, ' ', 0)}
	sw.line("boom\n")
	require.Error(t, sw.err)
	sw.line("ignored\n")
	assert.Contains(t, sw.err.Error(), "write output")
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced write failure")
}

func TestRootHelpers_HumanStatusAndPreRun(t *testing.T) {
	assert.False(t, isHumanStatusCommand(nil))
	parent := &cobra.Command{Use: "treasure-chest"}
	child := &cobra.Command{Use: "jewel"}
	parent.AddCommand(child)
	assert.True(t, isHumanStatusCommand(child))

	cmd := &cobra.Command{Use: "custom"}
	require.NoError(t, rootCmd.PersistentPreRunE(cmd, nil))
	assert.NotNil(t, cmd.Context())
	require.NoError(t, rootCmd.PersistentPostRunE(cmd, nil))
}

func TestValidateRuntimeDefaultParity_DetectsRuntimeDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("stale runtime copy"), 0o644))

	errs := validateRuntimeDefaultParity(root)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], `normative file "SKILL.md" differs`)
}

func TestRunCompile_CompileAllError(t *testing.T) {
	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = t.TempDir()

	err := runCompile(&cobra.Command{Use: "compile"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile: compile all")
}

func TestRunInstall_RejectsConflictingShimFlags(t *testing.T) {
	origNoShim := installNoShim
	origShimPath := installShimPath
	t.Cleanup(func() {
		installNoShim = origNoShim
		installShimPath = origShimPath
	})
	installNoShim = true
	installShimPath = filepath.Join(t.TempDir(), "SKILL.md")

	err := runInstall(&cobra.Command{Use: "install"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestPrintCheckSuccess_ClosedStdoutErrors(t *testing.T) {
	providers := map[string]string{"discovery": "brainstorming", "refinement": "openspec-explore", "execution": "sniper"}
	resolutions := map[string]slotResolution{
		"discovery":  {kind: slotResolutionSkillProvider},
		"refinement": {kind: slotResolutionSkillProvider},
		"execution":  {kind: slotResolutionNativeRole},
	}

	withClosedStdout(t, func() {
		require.Error(t, printCheckSuccess("/tmp/root", providers, resolutions, "epic"))
	})
}

func withClosedStdout(t *testing.T, fn func()) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "closed-stdout-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	fn()
}
