//go:build integration

package integration_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knownInvocationDebt lists documented command lines that still name the
// removed `strategist treasure-chest` namespace. They are tolerated here only
// until the contracts name the JEWELER / treasure-chest-tmp replacement; any
// other nonconformant line is new drift and fails this test.
var knownInvocationDebt = []string{"strategist treasure-chest "}

// TestE2E_CLI_InstalledRuntimeTextsMatchTheBinary runs the invocation
// conformance advisory through the installed binary against a freshly
// installed runtime: every command line agents copy from SKILL.md, the agent
// protocol, contracts and roles must be accepted by that same binary.
func TestE2E_CLI_InstalledRuntimeTextsMatchTheBinary(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	env := map[string]string{"STRATEGIST_TREASURE_CHEST_EXECUTABLE": ""}
	strategistDir := filepath.Join(workspace, ".strategist")
	install := runStrategistCLIWithEnv(t, workspace, env, "install", "--target", workspace, "--force", "--silent", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())

	check := runStrategistCLIWithEnv(t, workspace, env, "check", "--root", strategistDir, "--json")
	require.Equal(t, 0, check.exitCode, check.output())
	var result struct {
		Status   string   `json:"status"`
		Warnings []string `json:"warnings"`
	}
	require.NoError(t, json.Unmarshal([]byte(check.stdout), &result), check.output())

	assert.Equal(t, "ready", result.Status, "advisories never change readiness")
	var unexpected []string
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "reason=invocation_nonconformant") && !isKnownInvocationDebt(warning) {
			unexpected = append(unexpected, warning)
		}
	}
	assert.Empty(t, unexpected, "documented command lines the installed binary would reject")
	assert.Contains(t, strings.Join(result.Warnings, "\n"), "reason=support_binding_unready role=ranger feat=select_runbook")
}

func isKnownInvocationDebt(warning string) bool {
	for _, debt := range knownInvocationDebt {
		if strings.Contains(warning, "command=`"+debt) {
			return true
		}
	}
	return false
}
