//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderOnboardingDocsKeepNativeDiscoveryAuthority(t *testing.T) {
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "docs", "provider-extension.md"),
		filepath.Join(root, "docs", "configuration.md"),
		filepath.Join(root, "docs", "architecture", "strategist-concepts.md"),
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		for _, required := range []string{"Ranger", "live", "provider"} {
			if !strings.Contains(text, required) {
				t.Errorf("%s missing provider authority term %q", path, required)
			}
		}
	}
	fixture := filepath.Join(root, "internal", "embed", "defaults", "plugins", "fixtures", "minimal-provider")
	for _, name := range []string{"package.yaml", "adapter.yaml", "skill.yaml"} {
		if _, err := os.Stat(filepath.Join(fixture, name)); err != nil {
			t.Errorf("minimal provider fixture missing %s: %v", name, err)
		}
	}
}
