package policy_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pkgDigestA     = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	pkgDigestB     = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	adapterDigestA = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func TestEvaluateGrantAllowsExactDigestAndPermissionSubset(t *testing.T) {
	t.Parallel()

	result := policy.EvaluateGrant(policy.GrantRequest{
		PackageDigest: pkgDigestA,
		AdapterDigest: adapterDigestA,
		Requested: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
		},
	}, domain.PermissionGrant{
		PackageDigest: pkgDigestA,
		AdapterDigest: adapterDigestA,
		GrantedPermissions: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
		},
	})

	assert.True(t, result.Allowed)
	assert.Empty(t, result.Reasons)
}

func TestEvaluateGrantBlocksDigestDriftAndPermissionEscalation(t *testing.T) {
	t.Parallel()

	result := policy.EvaluateGrant(policy.GrantRequest{
		PackageDigest: pkgDigestB,
		AdapterDigest: adapterDigestA,
		Requested: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionNetworkAccess,
		},
	}, domain.PermissionGrant{
		PackageDigest: pkgDigestA,
		AdapterDigest: adapterDigestA,
		GrantedPermissions: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
		},
	})

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reasons, policy.DecisionReason{Code: "package_digest_mismatch", Detail: "grant is bound to a different package digest"})
	assert.Contains(t, result.Reasons, policy.DecisionReason{Code: "permission_escalation_requires_reconsent", Detail: "network.access"})
}

func TestEvaluateGrantRejectsInvalidDigestAndPermissionVocabulary(t *testing.T) {
	t.Parallel()

	result := policy.EvaluateGrant(policy.GrantRequest{
		PackageDigest: "sha256:not-a-real-digest",
		AdapterDigest: adapterDigestA,
		Requested:     []domain.PluginPermission{"filesystem.superuser"},
	}, domain.PermissionGrant{
		PackageDigest: pkgDigestA,
		AdapterDigest: adapterDigestA,
	})

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reasons, policy.DecisionReason{Code: "invalid_package_digest", Detail: "sha256:not-a-real-digest"})
	assert.Contains(t, result.Reasons, policy.DecisionReason{Code: "unknown_requested_permission", Detail: "filesystem.superuser"})
}

func TestEvaluateEnforcementReportsUnenforceableGrantedPermissions(t *testing.T) {
	t.Parallel()

	result := policy.EvaluateEnforcement(policy.EnforcementReport{
		ConnectorID: "current-runtime",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
		},
		Observed: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionNetworkAccess,
		},
	}, []domain.PluginPermission{
		domain.PluginPermissionReadWorkspace,
		domain.PluginPermissionNetworkAccess,
	})

	assert.False(t, result.Complete)
	assert.Contains(t, result.Unenforceable, domain.PluginPermissionNetworkAccess)
	assert.Contains(t, result.Observed, domain.PluginPermissionNetworkAccess)
	assert.Equal(t, "current-runtime", result.ConnectorID)
}

func TestEvaluateEnforcementCompleteWhenAllGrantedPermissionsAreEnforceable(t *testing.T) {
	t.Parallel()

	result := policy.EvaluateEnforcement(policy.EnforcementReport{
		ConnectorID: "native-test",
		Enforceable: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
			domain.PluginPermissionWriteAnalysis,
		},
	}, []domain.PluginPermission{
		domain.PluginPermissionWriteAnalysis,
		domain.PluginPermissionReadWorkspace,
	})

	require.True(t, result.Complete)
	assert.Empty(t, result.Unenforceable)
}
