package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

// LevelingAuthorityFixture is one on-disk LEVELING authority state and the
// reason code every surface (CLI and wizard) must report for it; an empty
// WantCode means the policy loads.
type LevelingAuthorityFixture struct {
	Name     string
	WantCode string
	Setup    func(t *testing.T, root string)
}

// LevelingReasonCode extracts the first `leveling_*` reason code from an
// error, or "" for nil.
func LevelingReasonCode(err error) string {
	if err == nil {
		return ""
	}
	return levelingCode.FindString(err.Error())
}

var levelingCode = regexp.MustCompile(`leveling_[a-z_]+`)

// EmbeddedLevelingDefaults returns the shipped leveling.yaml.
func EmbeddedLevelingDefaults(t *testing.T) []byte {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	return raw
}

// LevelingAuthorityFixtures lists the authority states both surfaces must
// classify identically. Each Setup writes into a .strategist root.
func LevelingAuthorityFixtures() []LevelingAuthorityFixture {
	return []LevelingAuthorityFixture{
		{Name: "missing", WantCode: leveling.ReasonLevelingPolicyMissing, Setup: func(*testing.T, string) {}},
		{Name: "unreadable override", WantCode: leveling.ReasonLevelingPolicyUnreadable, Setup: func(t *testing.T, root string) {
			require.NoError(t, os.MkdirAll(filepath.Join(root, "leveling.yaml"), 0o700))
		}},
		{Name: "stale manifest", WantCode: leveling.ReasonLevelingPolicyStale, Setup: func(t *testing.T, root string) {
			writeOverride(t, root, EmbeddedLevelingDefaults(t))
			writeManifest(t, root, 1, "stale-digest")
		}},
		{Name: "digestless manifest with shipped defaults", Setup: func(t *testing.T, root string) {
			writeOverride(t, root, EmbeddedLevelingDefaults(t))
			writeManifest(t, root, 0, "")
		}},
		{Name: "partial override with current manifest", Setup: func(t *testing.T, root string) {
			writeCurrentManifest(t, root)
			writeOverride(t, root, []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n"))
		}},
		{Name: "legacy marker without override", Setup: func(t *testing.T, root string) {
			require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 1\nmode: legacy\n"), 0o600))
		}},
	}
}

func writeOverride(t *testing.T, root string, raw []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, "leveling.yaml"), raw, 0o600))
}

func writeCurrentManifest(t *testing.T, root string) {
	t.Helper()
	policy, err := leveling.Parse(EmbeddedLevelingDefaults(t))
	require.NoError(t, err)
	writeManifest(t, root, policy.Version, policy.Digest())
}

func writeManifest(t *testing.T, root string, version int, digest string) {
	t.Helper()
	raw, err := json.Marshal(domain.InstallManifest{Schema: "strategist.install-manifest.v1", LevelingPolicyVersion: version, LevelingPolicyDigest: digest})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), raw, 0o600))
}

// PortableStrategistRoot creates a .strategist root under a path with a space
// and non-ASCII characters, the shape of a typical Windows user profile
// (`C:\Users\João Silva\...`). On the windows-latest CI job it is a real
// Windows path; elsewhere it still exercises separator and encoding handling.
func PortableStrategistRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Windows Client", "João Silva", "repo", ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o700))
	return root
}
