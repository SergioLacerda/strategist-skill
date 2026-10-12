package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskGatedPolicyWithRequirePredicatesUsesPolicyStatus(t *testing.T) {
	base := Policy{
		RequireWhen: []PolicyPredicate{PredicateMandatoryConstraintsPresent},
		SkipWhen:    []PolicyPredicate{PredicateInformationalOnly},
	}

	result := riskGatedPolicy(base, "high")
	assert.True(t, result.Enabled)
}

func TestReadVerificationMetadataRejectsMalformedAnalysis(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte("not frontmatter\n"), 0o600))

	_, err := ReadVerificationMetadata(dir)
	require.ErrorContains(t, err, "frontmatter is missing")
}
