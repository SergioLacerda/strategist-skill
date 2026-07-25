package i18n_test

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
)

// TestRuntimeBundlesHaveSameKeys verifies that ENRuntime and PTBRRuntime expose the same
// set of message keys and that no value is empty in either bundle.
func TestRuntimeBundlesHaveSameKeys(t *testing.T) {
	t.Parallel()

	en := reflect.ValueOf(i18n.ENRuntime)
	pt := reflect.ValueOf(i18n.PTBRRuntime)
	typ := en.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)
		enVal := en.Field(i).String()
		ptVal := pt.Field(i).String()

		assert.NotEmpty(t, enVal, "ENRuntime.%s must not be empty", field.Name)
		assert.NotEmpty(t, ptVal, "PTBRRuntime.%s must not be empty", field.Name)
	}
}

// TestReservedTokensAreNotInENRuntime ensures the EN runtime bundle does not embed
// Portuguese reserved input tokens (as standalone words) that belong in reserved.go.
func TestReservedTokensAreNotInENRuntime(t *testing.T) {
	t.Parallel()

	// These are matched as whole words to avoid false positives (e.g. "similar" contains "sim").
	forbidden := []string{"sim", "nao", "concordo", "faltou", "saque rapido"}
	en := reflect.ValueOf(i18n.ENRuntime)
	typ := en.Type()

	for i := range typ.NumField() {
		val := en.Field(i).String()
		for _, token := range forbidden {
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(token) + `\b`)
			assert.False(t, re.MatchString(val),
				"ENRuntime.%s must not embed reserved Portuguese token %q as a standalone word", typ.Field(i).Name, token)
		}
	}
}

// TestPTBRRuntimeHasPortugueseReviewGate verifies the PTBR bundle uses Portuguese gate prompt.
func TestPTBRRuntimeHasPortugueseReviewGate(t *testing.T) {
	t.Parallel()
	assert.Contains(t, i18n.PTBRRuntime.ApprovalGatePrompt, "sim / nao")
	assert.NotContains(t, i18n.ENRuntime.ApprovalGatePrompt, "sim / nao")
}

// TestENRuntimeHasDocumentationMaterializationSemantics verifies the EN bundle uses
// documentation-materialization terminology rather than legacy execution terms.
func TestENRuntimeHasDocumentationMaterializationSemantics(t *testing.T) {
	t.Parallel()

	forbidden := []string{"Authorize Sniper?", "Implement?", "implementation complete"}
	for _, bad := range forbidden {
		assert.NotContains(t, i18n.ENRuntime.ApprovalGatePrompt, bad,
			"ENRuntime.ApprovalGatePrompt must not contain legacy execution term %q", bad)
		assert.NotContains(t, i18n.ENRuntime.SniperDone, bad,
			"ENRuntime.SniperDone must not contain legacy execution term %q", bad)
	}

	assert.Contains(t, i18n.ENRuntime.SniperStart, "materialization")
	assert.Contains(t, i18n.ENRuntime.AdrGate, "Archive ADR")
}

// TestRuntimeMessagesToMap verifies ToMap exposes every field under its snake_case key.
func TestRuntimeMessagesToMap(t *testing.T) {
	t.Parallel()

	m := i18n.ENRuntime.ToMap()
	assert.Equal(t, i18n.ENRuntime.IntakeSummary, m["intake_summary"])
	assert.Equal(t, i18n.ENRuntime.RangerStart, m["ranger_start"])
	assert.Equal(t, i18n.ENRuntime.SniperDone, m["sniper_done"])
	assert.Equal(t, i18n.ENRuntime.ArtifactEntry, m["artifact_entry"])
	assert.NotEmpty(t, m)
}

// TestPTBRPhaseAnnouncementsFieldsNonEmpty verifies every phase_announcements field
// has a translated value — this is a distinct compiled-artifact field from
// content_by_lang (RuntimeMessages above), injected separately at compile time.
func TestPTBRPhaseAnnouncementsFieldsNonEmpty(t *testing.T) {
	t.Parallel()

	v := reflect.ValueOf(i18n.PTBRPhaseAnnouncements)
	typ := v.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		assert.NotEmpty(t, v.Field(i).String(), "PTBRPhaseAnnouncements.%s must not be empty", field.Name)
	}
}

// TestPhaseAnnouncementsMessagesToMap verifies ToMap exposes every field under its
// snake_case key, matching the phase_announcements.en conventions used in persona YAML.
func TestPhaseAnnouncementsMessagesToMap(t *testing.T) {
	t.Parallel()

	m := i18n.PTBRPhaseAnnouncements.ToMap()
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.DiscoveryStarting, m["discovery_starting"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.DiscoveryDone, m["discovery_done"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.RefinementStarting, m["refinement_starting"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.RefinementDone, m["refinement_done"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.ApprovalGateShown, m["approval_gate_shown"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.DocumentationStarting, m["documentation_starting"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.DocumentationTargetDone, m["documentation_target_done"])
	assert.Equal(t, i18n.PTBRPhaseAnnouncements.DocumentationDone, m["documentation_done"])
	assert.Len(t, m, reflect.TypeOf(i18n.PTBRPhaseAnnouncements).NumField())
}
