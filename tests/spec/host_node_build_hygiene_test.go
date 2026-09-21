//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenPrivateNodeTokens name the removed private-Node build branch: its
// build tag, its Node archive fetcher and its payload package. Every build
// now embeds only the OpenSpec bundle and runs it with the host Node.
var forbiddenPrivateNodeTokens = []string{"strategist_payload", "fetch-node-runtime.py", "nodepayload"}

// buildAndReleaseSurfaces lists every file that decides how Strategist is
// built, tested or released.
func buildAndReleaseSurfaces(t *testing.T, root string) map[string]string {
	t.Helper()
	surfaces := map[string]string{
		"Makefile":         readMakefileSystem(t, root),
		".goreleaser.yaml": readFile(t, filepath.Join(root, ".goreleaser.yaml")),
	}
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(workflows) == 0 {
		t.Fatalf("list workflows: %v (found %d)", err, len(workflows))
	}
	for _, path := range workflows {
		rel, _ := filepath.Rel(root, path)
		surfaces[rel] = readFile(t, path)
	}
	return surfaces
}

func TestBuildAndReleaseCarryNoPrivateNodeBranch(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	for name, content := range buildAndReleaseSurfaces(t, root) {
		for _, token := range forbiddenPrivateNodeTokens {
			if strings.Contains(content, token) {
				t.Errorf("%s references %q; the private-Node payload build was removed (docs/runbooks/standalone-runtime-hermeticity.md)", name, token)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "scripts", "fetch-node-runtime.py")); err == nil {
		t.Errorf("scripts/fetch-node-runtime.py must not exist: no build fetches a Node archive")
	}
}

// The smoke steps must prove the documented minimum host Node, not whatever
// the runner image happens to ship.
func TestStandaloneSmokeRunsOnTheMinimumHostNode(t *testing.T) {
	t.Parallel()
	workflow := readFile(t, filepath.Join(repoRoot(t), ".github", "workflows", "test.yml"))

	smokes := strings.Count(workflow, "./scripts/smoke-standalone-install.sh")
	pinned := strings.Count(workflow, "node-version: '20.19'")
	if smokes == 0 || pinned < smokes {
		t.Fatalf("each standalone smoke (%d) needs actions/setup-node with node-version '20.19' (found %d)", smokes, pinned)
	}
}
