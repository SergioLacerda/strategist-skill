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
