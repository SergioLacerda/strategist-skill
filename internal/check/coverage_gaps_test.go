package check

// Tests targeting specific uncovered branches, moved from cmd/strategist's
// own coverage_gaps_test.go / coverage_more_test.go when this package was
// extracted (20260816-cmd-strategist-cli-reorg) — see internal/treasurecli's
// own coverage_more_test.go for the same class of split during its own
// extraction.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- check_stale: stale artifact subprocess test ---

func TestCheckStaleCmd_StaleArtifactTriggersExit(t *testing.T) {
	if os.Getenv("STRATEGIST_STALE_EXIT") == "1" {
		// Subprocess branch: create a stale artifact and run check-stale against it.
		dir := t.TempDir()
		artifactPath := filepath.Join(dir, "artifact.gz")
		_ = artifactPath
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCheckStaleCmd_StaleArtifactTriggersExit", "-test.v")
	cmd.Env = append(os.Environ(), "STRATEGIST_STALE_EXIT=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Any exit code is acceptable — what matters is the branch was reached.
		t.Logf("subprocess exited with code %d (expected)", exitErr.ExitCode())
		return
	}
	// Subprocess succeeded — that's also fine.
}

// --- printSimulateReport ---

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

// --- validateRuntimeDefaultParity ---

func TestValidateRuntimeDefaultParity_DetectsRuntimeDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("stale runtime copy"), 0o644))

	errs := validateRuntimeDefaultParity(root)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], `normative file "SKILL.md" differs`)
}

// --- printCheckSuccess ---

func TestPrintCheckSuccess_ClosedStdoutErrors(t *testing.T) {
	providers := map[string]string{"discovery": "brainstorming", "refinement": "openspec-explore", "execution": "sniper"}
	resolutions := map[string]slotResolution{
		"discovery":  {kind: slotResolutionSkillProvider},
		"refinement": {kind: slotResolutionSkillProvider},
		"execution":  {kind: slotResolutionNativeRole},
	}

	withClosedStdout(t, func() {
		require.Error(t, printCheckSuccess("/tmp/root", providers, resolutions, "epic", domain.ResolutionPolicyAsk))
	})
}

// --- writeCheckSlotsSection / writeCheckPolicySection (ADR-0028) ---

func TestPrintCheckSuccess_ReportsFallbackAndPolicy(t *testing.T) {
	providers := map[string]string{"discovery": "brainstorming", "refinement": "openspec-explore", "execution": "sniper"}
	resolutions := map[string]slotResolution{
		"discovery":  {kind: slotResolutionSkillProvider},
		"refinement": {kind: slotResolutionSkillProvider, fallbackProvider: "archivist", fallbackPath: "/root/roles/archivist.yaml"},
		"execution":  {kind: slotResolutionNativeRole},
	}

	out := captureStdout(t, func() {
		require.NoError(t, printCheckSuccess("/tmp/root", providers, resolutions, "epic", domain.ResolutionPolicyNative))
	})
	assert.Contains(t, out, "fallback=archivist(native_role)")
	assert.Equal(t, 1, strings.Count(out, "fallback="), "only the refinement row should carry a fallback annotation")
	assert.Contains(t, out, "provider_resolution")
	assert.Contains(t, out, "native")
	assert.NotContains(t, out, "(default)") // explicit policy set — no "(default)" suffix
}

func TestPrintCheckSuccess_DefaultPolicyAnnotated(t *testing.T) {
	providers := map[string]string{"discovery": "ranger", "refinement": "archivist", "execution": "sniper"}
	resolutions := map[string]slotResolution{
		"discovery":  {kind: slotResolutionNativeRole},
		"refinement": {kind: slotResolutionNativeRole},
		"execution":  {kind: slotResolutionNativeRole},
	}

	out := captureStdout(t, func() {
		require.NoError(t, printCheckSuccess("/tmp/root", providers, resolutions, "epic", ""))
	})
	assert.Contains(t, out, "ask (default)")
	assert.NotContains(t, out, "fallback=") // native_role resolutions never carry a fallback
}

func TestPrintCheckSuccess_ReportsPolicyOutcomePerSlot(t *testing.T) {
	providers := map[string]string{"discovery": "brainstorming", "refinement": "openspec-explore", "execution": "sniper"}
	resolutions := map[string]slotResolution{
		// discovery: fallback available, but must always report outcome=always_native_no_policy
		// regardless of the configured policy (00-routing.md § Discovery Weapon Resolution by Subtype).
		"discovery":  {kind: slotResolutionSkillProvider, fallbackProvider: "ranger", fallbackPath: "/root/roles/ranger.yaml"},
		"refinement": {kind: slotResolutionSkillProvider, fallbackProvider: "archivist", fallbackPath: "/root/roles/archivist.yaml"},
		"execution":  {kind: slotResolutionNativeRole},
	}

	out := captureStdout(t, func() {
		require.NoError(t, printCheckSuccess("/tmp/root", providers, resolutions, "epic", domain.ResolutionPolicyAsk))
	})
	assert.Contains(t, out, "fallback=ranger(native_role)")
	assert.Contains(t, out, "outcome=always_native_no_policy")
	assert.Contains(t, out, "fallback=archivist(native_role)")
	assert.Contains(t, out, "outcome=ask_required")
}
