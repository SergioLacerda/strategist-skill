package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// forbiddenDials are the ways a test can leave the machine on its own. A test
// reaches a provider only through an injected transport or a loopback simulator.
var forbiddenDials = []string{"net.Dial(", "net.DialTimeout(", "http.Get(", "http.Post(", "http.PostForm(", "http.Head(", "http.DefaultClient", "http.DefaultTransport"}

// scannedDirs are the trees whose tests touch the provider integration.
var scannedDirs = []string{"internal/integration", "internal/install", "internal/mission/gatecriteria", "cmd/strategist"}

// isScannedTest reports whether a path is a test source this guard checks. This
// file itself and the opt-in live check (build tag "live") are the deliberate
// exceptions.
func isScannedTest(path string) bool {
	return strings.HasSuffix(path, "_test.go") && filepath.Base(path) != "hermetic_test.go" && filepath.Base(filepath.Dir(path)) != "live"
}

func assertNoDirectDial(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // G304: scanning repository test sources
	require.NoError(t, err)
	for _, pattern := range forbiddenDials {
		require.NotContains(t, string(raw), pattern, "%s dials the network directly with %s", path, pattern)
	}
}

// TestNoTestFileDialsTheNetworkOnItsOwn keeps the provider tests hermetic: it
// scans the test sources that touch the integration for a direct dial.
func TestNoTestFileDialsTheNetworkOnItsOwn(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range scannedDirs {
		require.NoError(t, filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && isScannedTest(path) {
				assertNoDirectDial(t, path)
			}
			return err
		}))
	}
}
