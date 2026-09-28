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
	require.NoError(t, os.Remove(filepath.Join(dir, "skills", "brainstorming", "SKILL.md")))
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
			assert.Equal(t, "blocked", b.Status, "discovery binding should be blocked due to entrypoint_payload_missing")
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

// TestCheckCmd_JSON_LanguageSurfacedFromActiveYAML confirms
// PreflightResult.Language mirrors active.yaml's language block directly, so
// an agent's bootstrap step can read chat/ui/docs/code language from this one
// `--json` call instead of separately, unpromptedly re-reading active.yaml
// (the enforcement gap behind 20260923-wizard-language-hardening).
func TestCheckCmd_JSON_LanguageSurfacedFromActiveYAML(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nlanguage:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.NotNil(t, result.Language)
	assert.Equal(t, "pt-BR", result.Language.UI)
	assert.Equal(t, "en", result.Language.Docs)
	assert.Equal(t, "pt-BR", result.Language.Chat)
	assert.Equal(t, "en", result.Language.Code)
}

// TestCheckCmd_JSON_LanguageAbsentWhenNotConfigured confirms Language is omitted
// (nil, and absent from the marshaled JSON via omitempty) for active.yaml
// files that predate the language block — no regression for the many fixtures
// across this package that don't declare one.
func TestCheckCmd_JSON_LanguageAbsentWhenNotConfigured(t *testing.T) {
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
	assert.Nil(t, result.Language)
	assert.NotContains(t, out, `"language"`)
}

// TestCheckCmd_JSON_ConfirmChatLanguageMismatchWarns confirms the mechanical
// enforcement point: an agent that acknowledges a chat language different
// from active.yaml's language.chat gets a real, blocking
// chat_language_mismatch warning instead of silently drifting.
func TestCheckCmd_JSON_ConfirmChatLanguageMismatchWarns(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nlanguage:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true
	checkConfirmChatLanguage = "en"

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.Error(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	assert.Contains(t, result.Warnings, "chat_language_mismatch: confirmed=en configured=pt-BR")
}

// TestCheckCmd_JSON_ConfirmChatLanguageMatchStaysReady confirms a matching
// acknowledgment never introduces a false-positive warning.
func TestCheckCmd_JSON_ConfirmChatLanguageMatchStaysReady(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nlanguage:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"),
		0o644,
	))
	checkRoot = dir
	checkJSON = true
	checkConfirmChatLanguage = "pt-BR"

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
	assert.Empty(t, result.Warnings)
}
