package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCmd_JSON_Success(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, domain.PreflightResultSchemaVersion, result.SchemaVersion)
	assert.Equal(t, "ready", result.Status)
	assert.Equal(t, dir, result.Identity.Root)
	assert.Equal(t, "epic", result.Identity.Mode)
	assert.Empty(t, result.Warnings)
	assert.Len(t, result.Bindings, 3)
	assert.Equal(t, "intake", result.Next, "docs/adr/0044 DEC-001: a ready result's Next must be a real phase token")
}

// TestCheckCmd_JSON_BlockedNextIsMessageNotPhaseToken confirms
// docs/adr/0044 DEC-001's other half: the blocked branch's Next stays the
// existing instructional message string, never a phase token like
// "intake" — only "ready" results get a phase token.
func TestCheckCmd_JSON_BlockedNextIsMessageNotPhaseToken(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte("id: [unterminated\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.Error(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	assert.Equal(t, "resolve the warnings below, then rerun `strategist check`", result.Next)
}

// TestCheckCmd_JSON_BindingsMatchResolvedProviderIDs is the D2/2.3 contract
// test: PreflightResult.Bindings must carry the exact provider ids
// check_slots.go's resolveSlotProvider resolved for each slot — sourced by
// reference, never recomputed independently.
func TestCheckCmd_JSON_BindingsMatchResolvedProviderIDs(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	wantProviders := map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-explore",
		"execution":  "sdd-ask",
	}
	got := map[string]string{}
	for _, b := range result.Bindings {
		got[b.Slot] = b.Provider
		assert.Equal(t, "ready", b.Status)
	}
	assert.Equal(t, wantProviders, got)
}

// TestCheckCmd_JSON_BindingStatusReflectsReadinessVector is the D6/task-2.5
// regression test: a slot whose provider resolved but whose readiness vector
// has a Blocked dimension (entrypoint id mismatch) must report
// binding.Status="blocked", not "ready" — status is derived from
// domain.PluginReadinessVector, not from mere provider-resolution success.
func TestCheckCmd_JSON_BindingStatusReflectsReadinessVector(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "skills", "brainstorming", "skill.yaml"),
		[]byte("id: not-brainstorming\nrisk_score: write_analysis\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.Error(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	for _, b := range result.Bindings {
		if b.Slot == "discovery" {
			assert.Equal(t, "blocked", b.Status, "discovery binding should be blocked due to entrypoint_id_mismatch")
		} else {
			assert.Equal(t, "ready", b.Status, "slot %s should remain ready", b.Slot)
		}
	}
}

func TestCheckCmd_JSON_BlockedStatusReportsWarnings(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	// Break the persona file so a validation warning is produced, exercising
	// the "blocked" branch without touching slot resolution.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "personas", "epic.yaml"),
		[]byte("id: [unterminated\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.Error(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	assert.NotEmpty(t, result.Warnings)
	assert.NotEmpty(t, result.Next)
}
