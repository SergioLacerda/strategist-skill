package mission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRunbookDestinationRuleUsesConfiguredOrLegacyRoot(t *testing.T) {
	configured, err := ResolveRunbookDestinationRule("m-1", ".analysis/runbooks")
	require.NoError(t, err)
	assert.Equal(t, ".analysis/runbooks", configured.CanonicalPath)
	assert.Equal(t, "docs/runbooks/m-1-runbook.md", configured.Fallback)

	legacy, err := ResolveRunbookDestinationRule("m-1", "")
	require.NoError(t, err)
	assert.Empty(t, legacy.CanonicalPath)
	assert.Equal(t, "docs/runbooks/m-1-runbook.md", legacy.Fallback)
}

func TestResolveRunbookDestinationRuleRejectsOutsideWorkspace(t *testing.T) {
	_, err := ResolveRunbookDestinationRule("m-1", "../runbooks")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runbook_destination_invalid")
}
