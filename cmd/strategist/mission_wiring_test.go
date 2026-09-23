package main

import (
	"bytes"
	"strings"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newWiredMissionCommand builds a fresh mission tree with the production
// dependencies from adapter_deps.go, so wiring tests never share flag state.
func newWiredMissionCommand() *cobra.Command {
	return missionadapter.New(missionLifecycleDependencies(), missionViewDependencies(), missionNormalizeDependencies(), missionReportUsageDependencies())
}

// executeMission runs `mission <args...>` on a fresh wired tree and returns
// its stdout.
func executeMission(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newWiredMissionCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestRootRegistersWiredMissionTree(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"mission"})
	require.NoError(t, err)
	assert.Equal(t, "mission", cmd.Name())
	assert.False(t, isHumanStatusCommand(cmd))
	for _, name := range []string{"start", "status", "submit", "context", "view", "normalize-openspec", "report-usage"} {
		sub, _, err := cmd.Find([]string{name})
		require.NoError(t, err, name)
		assert.Equal(t, name, sub.Name())
		assert.NotNil(t, sub.Flags().Lookup(cliutil.FlagRoot), name)
	}
}

func TestValidateMissionID(t *testing.T) {
	require.NoError(t, validateMissionID("valid-mission-123"))
	for _, test := range []struct{ id, want string }{
		{"", "--mission-id is required"},
		{"Invalid Mission", `--mission-id "Invalid Mission" is malformed (want lowercase letters, digits, and hyphens, e.g. 20260830-skill-gaps-triage)`},
		{"mission_123", "is malformed"},
		{"-leading-hyphen", "is malformed"},
	} {
		err := validateMissionID(test.id)
		require.Error(t, err, test.id)
		assert.Contains(t, err.Error(), test.want)
	}
}

// Every mission subcommand reports the same --mission-id text, behind its own
// command prefix (DEC-006).
func TestMissionSubcommandsShareMissionIDErrorText(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	prefixes := map[string]string{
		"start": "mission:", "status": "mission:", "submit": "mission:", "context": "mission:", "view": "mission:",
		"normalize-openspec": "mission normalize-openspec:", "report-usage": "mission report-usage:",
	}
	extra := map[string][]string{
		"normalize-openspec": {"--change-id", "change"},
		"report-usage":       {"--tokens-in", "1", "--tokens-out", "1"},
	}
	for sub, prefix := range prefixes {
		for id, want := range map[string]string{"": "--mission-id is required", "Bad Id": `--mission-id "Bad Id" is malformed`} {
			args := append([]string{sub, "--root", root, "--mission-id", id}, extra[sub]...)
			_, err := executeMission(t, args...)
			require.Error(t, err, "%s %q", sub, id)
			assert.True(t, strings.HasPrefix(err.Error(), prefix), "%s: %v", sub, err)
			assert.Contains(t, err.Error(), want, sub)
		}
	}
}
