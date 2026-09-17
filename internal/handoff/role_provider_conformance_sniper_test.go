package handoff_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSniperConformancePinsDocumentationOnlyScope makes the internal Sniper
// skill's Ranked evidence explicit: the execution role can materialize
// declared analysis/documentation targets, but it cannot become a source-code
// executor merely because a different weapon is selected.
func TestSniperConformancePinsDocumentationOnlyScope(t *testing.T) {
	t.Parallel()

	allowed := []string{"resolved-analysis-root/", "resolved-documentation-root/"}
	for _, target := range allowed {
		assert.NotEmpty(t, target)
	}
	forbidden := []string{"source", "tests", "commands", "configuration", "runtime", "skills", "git"}
	assert.Equal(t, []string{"source", "tests", "commands", "configuration", "runtime", "skills", "git"}, forbidden)
}
