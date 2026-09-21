package bundled

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The per-target embed files are compiled only with -tags strategist_payload, so
// ordinary builds never see a mistake in them. This test reads them as text and
// pins the three things that must agree: the build constraint, the embedded
// directory, and the target string registered at start-up.

var perTargetFile = regexp.MustCompile(`^embed_payload_([a-z0-9]+)_([a-z0-9]+)\.go$`)

func TestPerTargetEmbedFilesAgreeWithTheirName(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var seen []string
	for _, e := range entries {
		m := perTargetFile.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		goos, goarch := m[1], m[2]
		target := goos + "-" + goarch
		seen = append(seen, target)

		raw, readErr := os.ReadFile(filepath.Clean(e.Name()))
		require.NoError(t, readErr)
		body := string(raw)
		require.Contains(t, body, "//go:build strategist_payload && "+goos+" && "+goarch, e.Name()+" build constraint")
		require.Contains(t, body, "//go:embed nodepayload/"+target+"\n", e.Name()+" embedded directory")
		require.Contains(t, body, `registerTarget("`+target+`", nodePayload)`, e.Name()+" registered target")
		require.Equal(t, 1, strings.Count(body, "registerTarget("), e.Name()+" registers exactly one target")
	}
	require.NotEmpty(t, seen)
}

func TestEveryLockedNodeTargetHasAnEmbedFileAndViceVersa(t *testing.T) {
	raw, err := embed.DefaultsFS().Open("skills/openspec-propose/runtime.lock.yaml")
	require.NoError(t, err)
	defer raw.Close()
	var lock struct {
		Node struct {
			Targets map[string]struct {
				File string `yaml:"file"`
			} `yaml:"targets"`
		} `yaml:"node"`
	}
	require.NoError(t, yaml.NewDecoder(raw).Decode(&lock))
	require.NotEmpty(t, lock.Node.Targets)

	var locked, embedded []string
	for target := range lock.Node.Targets {
		locked = append(locked, target)
	}
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, e := range entries {
		if m := perTargetFile.FindStringSubmatch(e.Name()); m != nil {
			embedded = append(embedded, m[1]+"-"+m[2])
		}
	}
	sort.Strings(locked)
	sort.Strings(embedded)
	require.Equal(t, locked, embedded, "runtime.lock.yaml targets and embed_payload_<os>_<arch>.go files must match one to one")
}
