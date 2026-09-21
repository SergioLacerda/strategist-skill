//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rankedRuntimePackages are the packages that resolve and launch a Ranked
// provider runtime. The Windows standalone drift thread established that a
// Ranked runtime is never resolved from the client's machine; these packages
// are where that invariant can regress.
func rankedRuntimePackages() []string {
	return []string{
		filepath.Join("internal", "check"),
		filepath.Join("internal", "install"),
	}
}

// TestRankedRuntimeNeverResolvesOpenSpecFromPath fails when production code in
// the ranked runtime packages reintroduces PATH resolution for the provider
// executable. `runtimeenv.Command` is the PATH-resolving builder;
// `runtimeenv.PrivateCommand` is the contained one that requires an absolute,
// Strategist-resolved launcher.
//
// Resolving the *host Node* through exec.LookPath stays legal: host_node mode
// exists precisely to use a host Node, and it validates the result before use.
// Resolving `openspec` never is.
func TestRankedRuntimeNeverResolvesOpenSpecFromPath(t *testing.T) {
	t.Parallel()

	for _, pkg := range rankedRuntimePackages() {
		dir := filepath.Join(repoRoot(t), pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", pkg, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // G304: fixed repository path
			if err != nil {
				t.Fatalf("read %s/%s: %v", pkg, name, err)
			}
			body := string(raw)
			rel := filepath.ToSlash(filepath.Join(pkg, name))
			if strings.Contains(body, "runtimeenv.Command") {
				t.Errorf("%s calls runtimeenv.Command, which resolves the executable from PATH; "+
					"a Ranked runtime must launch through runtimeenv.PrivateCommand with an absolute path", rel)
			}
			if strings.Contains(body, `LookPath("openspec")`) || strings.Contains(body, `exec.LookPath("openspec")`) {
				t.Errorf("%s resolves the openspec executable from PATH; the embedded bundle is the only sanctioned source", rel)
			}
		}
	}
}
