package policy_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ClassifyWriteTarget ---

func TestClassifyWriteTarget_UnderBasePathIsAnalysis(t *testing.T) {
	t.Parallel()

	assert.Equal(t, domain.PluginPermissionWriteAnalysis, policy.ClassifyWriteTarget(".analysis", ".analysis/refined/mission/tasks.md"))
	assert.Equal(t, domain.PluginPermissionWriteAnalysis, policy.ClassifyWriteTarget(".analysis", ".analysis"))
}

func TestClassifyWriteTarget_RespectsConfiguredBasePathNotHardcodedAnalysis(t *testing.T) {
	t.Parallel()

	// A mission configured with a non-default base_path must still classify
	// correctly — this function must never hardcode ".analysis".
	assert.Equal(t, domain.PluginPermissionWriteAnalysis, policy.ClassifyWriteTarget("workspace/notes", "workspace/notes/mission/tasks.md"))
	assert.Equal(t, domain.PluginPermissionWriteSource, policy.ClassifyWriteTarget("workspace/notes", ".analysis/refined/mission/tasks.md"))
}

func TestClassifyWriteTarget_DocsPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, domain.PluginPermissionWriteDocs, policy.ClassifyWriteTarget(".analysis", "docs/adr/0001-example.md"))
	assert.Equal(t, domain.PluginPermissionWriteDocs, policy.ClassifyWriteTarget(".analysis", "docs"))
}

func TestClassifyWriteTarget_DefaultsToSource(t *testing.T) {
	t.Parallel()

	assert.Equal(t, domain.PluginPermissionWriteSource, policy.ClassifyWriteTarget(".analysis", "internal/check/check.go"))
	assert.Equal(t, domain.PluginPermissionWriteSource, policy.ClassifyWriteTarget(".analysis", "roles/sniper.yaml"))
}

func TestClassifyWriteTarget_EmptyBasePathNeverMatchesAnalysis(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, domain.PluginPermissionWriteAnalysis, policy.ClassifyWriteTarget("", "anything/at/all.md"))
}

func TestClassifyWriteTarget_DoesNotFalsePositiveOnPrefixCollision(t *testing.T) {
	t.Parallel()

	// ".analysis-archive" must not be treated as inside ".analysis" just
	// because it shares a string prefix.
	assert.NotEqual(t, domain.PluginPermissionWriteAnalysis, policy.ClassifyWriteTarget(".analysis", ".analysis-archive/notes.md"))
}

// --- EvaluateWrite ---

func TestEvaluateWrite_AllowsEnforceableAnalysisWrite(t *testing.T) {
	t.Parallel()

	report := policy.EnforcementReport{
		ConnectorID: "strategist-native",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
		},
	}

	decision := policy.EvaluateWrite(".analysis", ".analysis/refined/mission/tasks.md", report)
	require.True(t, decision.Allowed)
	assert.Equal(t, domain.PluginPermissionWriteAnalysis, decision.Permission)
	assert.Contains(t, decision.Reason, "enforceable")
}

func TestEvaluateWrite_DeniesDocsWriteOutsideEnforceableSet(t *testing.T) {
	t.Parallel()

	// Mirrors NativeRuntimeConnector.Observe's actual enforceable set today
	// (K03): only [ReadWorkspace, WriteAnalysis] — WriteDocs is not in it.
	report := policy.EnforcementReport{
		ConnectorID: "strategist-native",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
		},
	}

	decision := policy.EvaluateWrite(".analysis", "docs/adr/0001-example.md", report)
	require.False(t, decision.Allowed, "a write outside the enforceable allowlist must be denied by default")
	assert.Equal(t, domain.PluginPermissionWriteDocs, decision.Permission)
	assert.Contains(t, decision.Reason, "not in connector")
}

func TestEvaluateWrite_DeniesSourceWriteOutsideEnforceableSet(t *testing.T) {
	t.Parallel()

	report := policy.EnforcementReport{
		ConnectorID: "strategist-native",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
		},
	}

	decision := policy.EvaluateWrite(".analysis", "internal/check/check.go", report)
	require.False(t, decision.Allowed)
	assert.Equal(t, domain.PluginPermissionWriteSource, decision.Permission)
}

func TestEvaluateWrite_DeniesEverythingWhenConnectorEnforcesNothing(t *testing.T) {
	t.Parallel()

	// Mirrors UnsupportedConnector.Observe: Enforceable is always empty.
	report := policy.EnforcementReport{ConnectorID: "current-runtime", Limitations: []string{"enforcement_not_supported"}}

	analysis := policy.EvaluateWrite(".analysis", ".analysis/refined/mission/tasks.md", report)
	assert.False(t, analysis.Allowed)

	docs := policy.EvaluateWrite(".analysis", "docs/adr/0001-example.md", report)
	assert.False(t, docs.Allowed)

	source := policy.EvaluateWrite(".analysis", "internal/check/check.go", report)
	assert.False(t, source.Allowed)
}

func TestEvaluateWrite_AllowsWhenConnectorEnforcesEverything(t *testing.T) {
	t.Parallel()

	report := policy.EnforcementReport{
		ConnectorID: "fully-enforced-runtime",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
			domain.PluginPermissionWriteDocs,
			domain.PluginPermissionWriteSource,
		},
	}

	for _, target := range []string{
		".analysis/refined/mission/tasks.md",
		"docs/adr/0001-example.md",
		"internal/check/check.go",
	} {
		decision := policy.EvaluateWrite(".analysis", target, report)
		assert.True(t, decision.Allowed, "expected %q to be allowed", target)
	}
}
