//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const e2eRankerResult = "## mission_objective\nx\n## known_facts\n- f\n## confidence_summary\nc\n## handoff\nh\n"

func startDiscoveryMission(t *testing.T, workspace, missionID string) {
	t.Helper()
	for _, args := range [][]string{
		{"mission", "start", "--mission-id", missionID},
		{"mission", "submit", "--mission-id", missionID, "--event", "bootstrap_done"},
		{"mission", "submit", "--mission-id", missionID, "--event", "intake_done"},
	} {
		res := runStrategistCLI(t, workspace, args...)
		require.Equal(t, 0, res.exitCode, res.output())
	}
}

func artifactText(t *testing.T, workspace, missionID string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(workspace, ".analysis", "pending", missionID+"-analysis.md"))
	require.NoError(t, err)
	return string(raw)
}

func fakeCodexEnv(t *testing.T, fail bool) map[string]string {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\noutput=\"\"\nwhile [ \"$#\" -gt 0 ]; do if [ \"$1\" = \"--output-last-message\" ]; then shift; output=\"$1\"; fi; shift; done\ncat >/dev/null\n"
	if fail {
		script += "echo boom >&2\nexit 1\n"
	} else {
		script += "printf '%s\\n' '## mission_objective' 'x' '## known_facts' '- f' '## confidence_summary' 'c' '## handoff' 'h' > \"$output\"\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(bin, "codex"), []byte(script), 0o700))
	return map[string]string{"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH"), "CODEX_HOME": t.TempDir()}
}

// TestE2E_InstalledBinaryHonorsTheInvocationAndAdapterContract exercises the
// built binary, not staged source: a stale binary that omits a required field
// fails here instead of being reported as host conformance.
func TestE2E_InstalledBinaryHonorsTheInvocationAndAdapterContract(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())

	t.Run("emitted request carries the required envelope and completes as the current host", func(t *testing.T) {
		startDiscoveryMission(t, workspace, "adapter-host")
		emitted := runStrategistCLI(t, workspace, "mission", "invoke", "--mission-id", "adapter-host", "--slot", "discovery", "--json")
		require.Equal(t, 0, emitted.exitCode, emitted.output())
		var request map[string]any
		require.NoError(t, json.Unmarshal([]byte(emitted.stdout), &request))
		for _, field := range []string{"protocol", "request_id", "mission_id", "weapon", "binding_digest", "source_digest", "entrypoint", "payload", "nonce"} {
			require.Contains(t, request, field, "installed binary omits a field the checked-out contract requires: stale binary or artifact drift")
		}
		requestID, _ := request["request_id"].(string) //nolint:errcheck // asserted below.
		require.NotEmpty(t, requestID)
		forged, err := json.Marshal(map[string]string{"request_id": requestID, "result": "---\nexecution_adapter: codex_child\ncapability_isolation: verified\n---\n\n" + e2eRankerResult})
		require.NoError(t, err)

		done := runStrategistCLIWithInput(t, workspace, nil, string(forged), "mission", "complete", "--mission-id", "adapter-host", "--request-id", requestID, "--json")
		require.Equal(t, 0, done.exitCode, done.output())
		got := artifactText(t, workspace, "adapter-host")
		assert.Contains(t, got, "execution_adapter: current_host_adapter")
		assert.Contains(t, got, "capability_isolation: unverified")
		assert.NotContains(t, got, "codex_child")

		replay := runStrategistCLIWithInput(t, workspace, nil, string(forged), "mission", "complete", "--mission-id", "adapter-host", "--request-id", requestID, "--json")
		assert.NotEqual(t, 0, replay.exitCode)
		assert.Contains(t, replay.output(), "invocation_replay")
	})

	t.Run("CLI-owned Codex child is recorded as codex_child and stays unverified", func(t *testing.T) {
		startDiscoveryMission(t, workspace, "adapter-child")
		res := runStrategistCLIWithEnv(t, workspace, fakeCodexEnv(t, false), "mission", "invoke", "--mission-id", "adapter-child", "--slot", "discovery", "--host", "codex", "--context", "evaluate", "--json")
		require.Equal(t, 0, res.exitCode, res.output())
		got := artifactText(t, workspace, "adapter-child")
		assert.Contains(t, got, "execution_adapter: codex_child")
		assert.Contains(t, got, "child_policy_id: strategist-child-policy/v1:codex:")
		assert.Contains(t, got, "capability_isolation: unverified")
	})

	t.Run("a committed child cannot be completed through the current-host channel", func(t *testing.T) {
		startDiscoveryMission(t, workspace, "adapter-mismatch")
		failed := runStrategistCLIWithEnv(t, workspace, fakeCodexEnv(t, true), "mission", "invoke", "--mission-id", "adapter-mismatch", "--slot", "discovery", "--host", "codex", "--context", "evaluate", "--json")
		require.NotEqual(t, 0, failed.exitCode)
		requestID := committedRequestID(t, workspace, "adapter-mismatch")
		body, err := json.Marshal(map[string]string{"request_id": requestID, "result": e2eRankerResult})
		require.NoError(t, err)

		res := runStrategistCLIWithInput(t, workspace, nil, string(body), "mission", "complete", "--mission-id", "adapter-mismatch", "--request-id", requestID, "--json")

		assert.NotEqual(t, 0, res.exitCode)
		assert.Contains(t, res.output(), "invocation_adapter_mismatch")
		assert.NoFileExists(t, filepath.Join(workspace, ".analysis", "pending", "adapter-mismatch-analysis.md"))
	})
}

// committedRequestID finds the request a failed child run left committed.
func committedRequestID(t *testing.T, workspace, missionID string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(workspace, ".strategist", "missions", "invocations", "inv_*.json"))
	require.NoError(t, err)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		if strings.Contains(string(raw), fmt.Sprintf("%q", missionID)) && strings.Contains(string(raw), `"codex_child"`) {
			return strings.TrimSuffix(filepath.Base(file), ".json")
		}
	}
	require.FailNow(t, "no committed child request found for "+missionID)
	return ""
}
