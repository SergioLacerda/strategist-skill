//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// providerBoundaryDocs are the operator-facing documents that must describe the
// discovery slot as owned by native Ranger, not as externally selectable.
func providerBoundaryDocs() []string {
	return []string{"docs/configuration.md", "docs/provider-extension.md"}
}

// TestProviderDocsStateNativeRangerDiscoveryBoundary fails when the provider
// documentation stops naming the native_role_authority rejection, or describes
// discovery as replaceable by an external provider.
func TestProviderDocsStateNativeRangerDiscoveryBoundary(t *testing.T) {
	t.Parallel()

	stale := []string{
		"discovery provider can replace ranger",
		"external discovery provider replaces ranger",
		"select an external discovery provider",
	}
	for _, rel := range providerBoundaryDocs() {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := strings.Join(strings.Fields(strings.ToLower(string(raw))), " ")
		if !strings.Contains(text, "native_role_authority") {
			t.Errorf("%s must name the native_role_authority rejection for discovery", rel)
		}
		if !strings.Contains(text, "native ranger") {
			t.Errorf("%s must state that discovery remains owned by native Ranger", rel)
		}
		for _, phrase := range stale {
			if strings.Contains(text, phrase) {
				t.Errorf("%s contains stale wording %q", rel, phrase)
			}
		}
	}
}
