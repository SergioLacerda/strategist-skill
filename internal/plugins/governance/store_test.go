package governance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	policy := domain.TrustPolicy{
		SchemaVersion: domain.TrustPolicySchemaVersion,
		Revision:      "r1", TrustedPublishers: []string{"acme"},
		TrustedSources: []string{"embedded"}, MinimumConformance: "C1",
	}
	grants := domain.PermissionGrantFile{
		SchemaVersion: domain.PermissionGrantFileSchemaVersion,
		Grants: []domain.PermissionGrant{{
			ID: "grant-1", PackageDigest: digest("1"), AdapterDigest: digest("2"),
			GrantedPermissions: []domain.PluginPermission{domain.PluginPermissionReadWorkspace},
		}},
	}
	require.NoError(t, Save(root, policy, grants))
	gotPolicy, gotGrants, err := Load(root)
	require.NoError(t, err)
	require.Equal(t, policy, gotPolicy)
	require.Equal(t, grants, gotGrants)
	got, ok := FindGrant(gotGrants, digest("1"), digest("2"))
	require.True(t, ok)
	require.Equal(t, "grant-1", got.ID)
}

func TestLoadMissingFilesIsLegacyCompatible(t *testing.T) {
	policy, grants, err := Load(t.TempDir())
	require.NoError(t, err)
	require.Empty(t, policy)
	require.Empty(t, grants)
}

func TestLoadMalformedGovernanceFailsClosed(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, TrustPolicyFileName), []byte("revision: ["), 0o644))
	_, _, err := Load(root)
	require.ErrorContains(t, err, "parse trust-policy.yaml")
}

func TestSaveRejectsUnknownGrant(t *testing.T) {
	root := t.TempDir()
	grant := domain.PermissionGrant{ID: "grant-1", PackageDigest: digest("1"), AdapterDigest: digest("2"), GrantedPermissions: []domain.PluginPermission{"unknown"}}
	err := Save(root, domain.TrustPolicy{Revision: "r1"}, domain.PermissionGrantFile{Grants: []domain.PermissionGrant{grant}})
	require.ErrorContains(t, err, "unknown permission")
}

func digest(s string) string {
	return "sha256:" + s + "000000000000000000000000000000000000000000000000000000000000000"
}
