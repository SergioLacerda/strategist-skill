package leveling

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func authorityDefaults(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	return raw
}

func writeAuthorityManifest(t *testing.T, root string, version int, digest string) {
	t.Helper()
	raw, err := json.Marshal(domain.InstallManifest{
		Schema:                "strategist.install-manifest.v1",
		LevelingPolicyVersion: version,
		LevelingPolicyDigest:  digest,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), raw, 0o600))
}

func TestLoadAuthorizedUsesCurrentManifestAndMergesOverride(t *testing.T) {
	defaults := authorityDefaults(t)
	defaultPolicy, err := Parse(defaults)
	require.NoError(t, err)
	root := t.TempDir()
	writeAuthorityManifest(t, root, defaultPolicy.Version, defaultPolicy.Digest())

	effective, decision, err := LoadAuthorized(root, defaults, []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n"), "leveling.yaml")
	require.NoError(t, err)
	require.Equal(t, AuthorityCurrent, decision.Class)
	require.True(t, decision.ManifestLoaded)
	require.Equal(t, "medium", effective.Policy.Defaults.Roles["ranger"].Effort)
	require.NotEmpty(t, effective.Policy.Providers["CODEX"].Models)
}

func TestLoadAuthorizedFailsClosedForManifestErrors(t *testing.T) {
	defaults := authorityDefaults(t)
	tests := []struct {
		name  string
		write func(t *testing.T, path string)
		match string
		class AuthorityClass
	}{
		{
			name: "malformed",
			write: func(t *testing.T, path string) {
				require.NoError(t, os.WriteFile(path, []byte("not-json"), 0o600))
			},
			match: "leveling_policy_stale",
			class: AuthorityStale,
		},
		{
			name: "unreadable path",
			write: func(t *testing.T, path string) {
				require.NoError(t, os.Mkdir(path, 0o700))
			},
			match: "leveling_policy_stale",
			class: AuthorityUnreadable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.write(t, filepath.Join(root, domain.InstallManifestRelPath))
			_, decision, err := LoadAuthorized(root, defaults, defaults, "leveling.yaml")
			require.ErrorContains(t, err, tt.match)
			require.Equal(t, tt.class, decision.Class)
		})
	}
}

func TestLoadAuthorizedRequiresExplicitLegacyMarker(t *testing.T) {
	defaults := authorityDefaults(t)
	override := []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n")
	root := t.TempDir()
	_, decision, err := LoadAuthorized(root, defaults, override, "leveling.yaml")
	require.ErrorContains(t, err, "leveling_policy_missing")
	require.Equal(t, AuthorityMissing, decision.Class)

	require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 1\nmode: legacy\n"), 0o600))
	_, decision, err = LoadAuthorized(root, defaults, override, "leveling.yaml")
	require.NoError(t, err)
	require.Equal(t, AuthorityLegacyCompatible, decision.Class)
}

func TestLoadAuthorizedClassifiesDigestlessManifestAsMissingWhenOverrideDiffers(t *testing.T) {
	defaults := authorityDefaults(t)
	override := []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n")
	root := t.TempDir()
	writeAuthorityManifest(t, root, 1, "")
	_, decision, err := LoadAuthorized(root, defaults, override, "leveling.yaml")
	require.ErrorContains(t, err, "leveling_policy_missing")
	require.Equal(t, AuthorityMissing, decision.Class)
}

func TestLoadAuthorizedRejectsStaleManifest(t *testing.T) {
	defaults := authorityDefaults(t)
	root := t.TempDir()
	writeAuthorityManifest(t, root, 1, "stale")
	_, decision, err := LoadAuthorized(root, defaults, defaults, "leveling.yaml")
	require.ErrorContains(t, err, "leveling_policy_stale")
	require.Equal(t, AuthorityStale, decision.Class)
}

// The stale-manifest diagnostic names the embedded authority as the expected
// side and the install manifest as the observed one, and points at the remedy.
func TestStaleManifestDiagnosticNamesEmbeddedPolicyAsExpected(t *testing.T) {
	defaults := authorityDefaults(t)
	policy, err := Parse(defaults)
	require.NoError(t, err)
	root := t.TempDir()
	writeAuthorityManifest(t, root, policy.Version, "stale-digest")

	_, _, err = LoadAuthorized(root, defaults, defaults, "leveling.yaml")
	require.Error(t, err)
	message := err.Error()
	expectedAt, observedAt := strings.Index(message, "expected"), strings.Index(message, "observed")
	require.True(t, expectedAt >= 0 && observedAt > expectedAt, message)
	require.Contains(t, message[expectedAt:observedAt], policy.Digest(), "expected must be the embedded authority")
	require.Contains(t, message[observedAt:], "stale-digest", "observed must be the install manifest")
	require.Contains(t, message, "strategist upgrade")
}
