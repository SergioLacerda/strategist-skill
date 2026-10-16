//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const e2eRankerResult = "## mission_objective\nx\n## known_facts\n- f\n## confidence_summary\nc\n## handoff\nh\n"

func startDiscoveryMission(t *testing.T, workspace, missionID string) {
	t.Helper()
	for _, args := range [][]string{
		{"mission", "start", "--mission-id", missionID},
		{"mission", "submit", "--mission-id", missionID, "--event", "bootstrap_done"},
	} {
		res := runStrategistCLI(t, workspace, args...)
		require.Equal(t, 0, res.exitCode, res.output())
	}
	recordE2ERoute(t, workspace, missionID)
	res := runStrategistCLI(t, workspace, "mission", "submit", "--mission-id", missionID, "--event", "intake_done")
	require.Equal(t, 0, res.exitCode, res.output())
}

func artifactText(t *testing.T, workspace, missionID string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(workspace, ".analysis", "pending", missionID+"-analysis.md"))
	require.NoError(t, err)
	return string(raw)
}

func fakeCodexEnv(t *testing.T, fail bool) map[string]string {
	t.Helper()
	failure := `		fmt.Fprintln(os.Stderr, "boom")
		_ = output
		os.Exit(1)`
	if fail {
		// Keep the failure path identical to the old shell stub.
	} else {
		failure = `		if output == "" {
			fmt.Fprintln(os.Stderr, "missing output path")
			os.Exit(2)
		}
		if err := os.WriteFile(output, []byte(result), 0o600); err != nil {
			panic(err)
		}`
	}
	source := `package main

import (
	"fmt"
	"io"
	"os"
)

const result = ` + strconv.Quote(e2eRankerResult) + `

func main() {
	output := ""
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--output-last-message" && i+1 < len(os.Args) {
			output = os.Args[i+1]
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
` + failure + `
}
`
	stub := testutil.BuildGoExecutable(t, "codex", source)
	return map[string]string{"PATH": testutil.PrependPath(filepath.Dir(stub)), "CODEX_HOME": t.TempDir()}
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
