//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// providerBoundaryDocs are the operator-facing documents that must describe the
// discovery route as owned by Ranger and fulfilled through a selected Weapon.
func providerBoundaryDocs() []string {
	return []string{"docs/configuration.md", "docs/provider-extension.md"}
}

// TestProviderDocsStateRangerWeaponBoundary fails when provider documentation
// describes discovery as a replaceable native fallback instead of a required,
// fail-closed Ranger Weapon invocation.
func TestProviderDocsStateNativeRangerDiscoveryBoundary(t *testing.T) {
	t.Parallel()

	stale := []string{
		"discovery provider can replace ranger",
		"external discovery provider replaces ranger",
		"select an external discovery provider",
		"native_role_authority",
	}
	for _, rel := range providerBoundaryDocs() {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := strings.Join(strings.Fields(strings.ToLower(string(raw))), " ")
		if !strings.Contains(text, "native ranger") {
			t.Errorf("%s must state that discovery remains owned by native Ranger", rel)
		}
		if !strings.Contains(text, "role_invocation_failed") || !strings.Contains(text, "normalize") {
			t.Errorf("%s must require fail-closed Ranger Weapon invocation and normalization", rel)
		}
		for _, phrase := range stale {
			if strings.Contains(text, phrase) {
				t.Errorf("%s contains stale wording %q", rel, phrase)
			}
		}
	}
}
