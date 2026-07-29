//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EFeatureFilesCoverHappyPathContracts(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		filepath.Join(testDir(t), "specs", "e2e-happy-path.feature"): []string{
			"Ranger consults treasure chests",
			"Ranger records scope observations",
			"Archivist runs opportunity attack",
			"approval gate",
		},
		filepath.Join(testDir(t), "specs", "e2e-approval-gate.feature"): []string{
			"mission result is documentation_applied",
			"analysis_delivered",
			"Sniper is not invoked",
		},
		filepath.Join(testDir(t), "specs", "e2e-treasure-chests.feature"): []string{
			"treasure chests",
			"treasure_chests=none",
		},
		filepath.Join(testDir(t), "specs", "e2e-opportunity-attack.feature"): []string{
			"only Archivist performs Opportunity Attack",
			"Critical Hit remains responsible",
			"approval gate",
		},
		filepath.Join(testDir(t), "specs", "e2e-install-compile.feature"): []string{
			"customized active.yaml",
			"--force",
			"check-stale reports the compiled config as fresh",
		},
	}

	for path, needles := range files {
		content := readFile(t, path)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing %q", path, needle)
			}
		}
	}
}

// TestNoRootLevelProviderLookupInCode ensures resolver-facing code never references
// a root-level .strategist/<provider>/skill.yaml without the skills/ subdirectory.
// This guards the canonical runtime layout contract: all external provider manifests
// must resolve from .strategist/skills/<provider>/skill.yaml.
