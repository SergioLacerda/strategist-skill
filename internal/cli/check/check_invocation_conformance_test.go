package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCLITree mirrors the shape of the real CLI: a required flag declared
// through the usage text, one marked with MarkFlagRequired, and a persistent
// root flag every subcommand inherits.
func fakeCLITree(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "strategist"}
	root.PersistentFlags().String("root", "", "path to .strategist/ root")
	missionCmd := &cobra.Command{Use: "mission"}
	complete := &cobra.Command{Use: "complete", RunE: func(*cobra.Command, []string) error { return nil }}
	complete.Flags().String("mission-id", "", "mission identifier (required)")
	complete.Flags().String("request-id", "", "pending invocation request identifier (required)")
	complete.Flags().Bool("json", false, "emit JSON")
	missionCmd.AddCommand(complete)
	runbook := &cobra.Command{Use: "runbook"}
	selectCmd := &cobra.Command{Use: "select", RunE: func(*cobra.Command, []string) error { return nil }}
	selectCmd.Flags().StringArray("signal", nil, "mission signal")
	selectCmd.Flags().String("mission", "", "mission identifier")
	selectCmd.Flags().String("format", "table", "output format")
	require.NoError(t, selectCmd.MarkFlagRequired("mission"))
	runbook.AddCommand(selectCmd)
	root.AddCommand(missionCmd, runbook)
	return root
}

func writeRuntimeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestInvocationConformanceReportsMissingRequiredFlag(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "SKILL.md", "Return it to `strategist mission complete --request-id <id> --json`.\n")

	got := invocationConformanceAdvisories(root, fakeCLITree(t))

	require.Len(t, got, 1)
	assert.Contains(t, got[0], "reason=invocation_nonconformant")
	assert.Contains(t, got[0], "file=SKILL.md")
	assert.Contains(t, got[0], "line=1")
	assert.Contains(t, got[0], "missing required flag --mission-id")
}

func TestInvocationConformanceReportsRequiredFlagMarkedByAnnotation(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "roles/ranger.yaml", "invoke: \"strategist runbook select --format json --signal <signal> [--signal <signal> ...]\"\n")

	got := invocationConformanceAdvisories(root, fakeCLITree(t))

	require.Len(t, got, 1)
	assert.Contains(t, got[0], "file=roles/ranger.yaml")
	assert.Contains(t, got[0], "missing required flag --mission")
}

func TestInvocationConformanceReportsUnknownFlagAndCommand(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "agent-protocol.md", strings.Join([]string{
		"Run `strategist mission complete --mission-id <id> --request-id <id> --bogus`.",
		"Run `strategist mission vanish --mission-id <id>`.",
	}, "\n"))

	got := invocationConformanceAdvisories(root, fakeCLITree(t))

	require.Len(t, got, 2)
	assert.Contains(t, got[0], "unknown flag --bogus")
	assert.Contains(t, got[1], "unknown command \"mission vanish\"")
}

func TestInvocationConformanceAcceptsConformingAndIllustrativeLines(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "contracts/narrative/03-discovery.md", strings.Join([]string{
		"Mentioning `strategist mission complete` without flags is prose, not an invocation.",
		"`strategist mission complete --mission-id <id> --request-id <id> --json < completion.json`",
		"```bash",
		"strategist runbook select --mission <mission_id> --format json --signal a --signal b --root .strategist",
		"```",
		"The `strategist` binary is installed.",
		"Use `strategist mission start|status|submit --mission-id <id>`.",
		"Decided by `strategist (local, deterministic)` only.",
		"Run `strategist runbook; check (layout skew)`.",
		"Then `strategist mission complete --mission-id <id> --request-id <id>; strategist check`.",
	}, "\n"))

	assert.Empty(t, invocationConformanceAdvisories(root, fakeCLITree(t)))
}

func TestPreflightAdvisoriesIncludeInvocationConformance(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "SKILL.md", "`strategist mission complete --request-id <id>`\n")
	previous := conformanceTree
	conformanceTree = fakeCLITree(t)
	t.Cleanup(func() { conformanceTree = previous })

	got := strings.Join(preflightAdvisories(root), "\n")

	assert.Contains(t, got, "reason=invocation_nonconformant")
}

func TestInvocationConformanceWithoutTreeReportsNothing(t *testing.T) {
	root := t.TempDir()
	writeRuntimeFile(t, root, "SKILL.md", "`strategist mission complete --json`\n")

	assert.Empty(t, invocationConformanceAdvisories(root, nil))
}
