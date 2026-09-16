package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckCmd_JSON_IndexYAMLNotFoundIsAdvisoryOnly is the Onda 1
// regression test for preflight.yaml's index_yaml_not_found condition
// (20260916-preflight-cli-enforcement-wave3 Task A.1): removing index.yaml
// from an otherwise-valid root must surface a warning without flipping
// PreflightResult.Status or `strategist check`'s exit code.
func TestCheckCmd_JSON_IndexYAMLNotFoundIsAdvisoryOnly(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "index.yaml")))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "reason=index_yaml_not_found")
}

// TestCheckCmd_JSON_CompiledArtifactCorruptIsAdvisoryOnly covers Task A.1's
// other condition: a present-but-corrupt .compiled/.domain.gz is reported,
// non-blocking, and takes priority over index_yaml_not_found (the compiled
// artifact is preferred when present, per domainIndexAdvisories).
func TestCheckCmd_JSON_CompiledArtifactCorruptIsAdvisoryOnly(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(compiledDir, ".domain.gz"), []byte("not a gzip stream"), 0o644))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "reason=compiled_artifact_corrupt")
}

// TestCheckCmd_JSON_CompiledArtifactValidHasNoAdvisory confirms a
// well-formed compiled artifact produces no advisory at all, even though
// index.yaml is absent — the compiled artifact is a valid substitute.
func TestCheckCmd_JSON_CompiledArtifactValidHasNoAdvisory(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "index.yaml")))
	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o755))
	testutil.WriteGzJSON(t, filepath.Join(compiledDir, ".domain.gz"), map[string]any{"schema": "strategist-compiled-domain/1.0"})
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	assert.Empty(t, result.Warnings)
}

// TestCheckCmd_JSON_DirectivesMissingIsAdvisoryOnly covers Task A.2's
// non-blocking condition.
func TestCheckCmd_JSON_DirectivesMissingIsAdvisoryOnly(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "templates", "domain", "directives", "core.yaml")))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "reason=directives_missing")
}

// TestCheckCmd_JSON_AdvisoriesCoexistWithBlockingWarnings confirms an
// advisory doesn't get lost or double-count status when a real blocking
// warning is also present — both must appear in Warnings, and Status must
// still be "blocked" (driven by the blocking warning alone).
func TestCheckCmd_JSON_AdvisoriesCoexistWithBlockingWarnings(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "index.yaml")))
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
	assert.Contains(t, strings.Join(result.Warnings, "\n"), "reason=index_yaml_not_found")
}

// TestCheckCmd_JSON_IdentityFilesMissingEmitsBlockedEnvelope is the
// printPreflightJSONBlocked regression test: check_identity.go's
// hard-blocking identity_files_missing error must still produce a
// structured PreflightResult under --json, not a bare error string.
func TestCheckCmd_JSON_IdentityFilesMissingEmitsBlockedEnvelope(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "templates", "domain", "identity")))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.Error(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, domain.PreflightResultSchemaVersion, result.SchemaVersion)
	assert.Equal(t, "blocked", result.Status)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "reason=identity_files_missing")
	assert.NotEmpty(t, result.Next)
	assert.Empty(t, result.Bindings, "identity check fires before slot resolution — Bindings must not be populated")
}
