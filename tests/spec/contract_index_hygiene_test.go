//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type contractTestFixture struct {
	Subject string `yaml:"subject"`
}

type contractIndexFixture struct {
	Machine struct {
		AlwaysLoad []string            `yaml:"always_load"`
		ByPhase    map[string][]string `yaml:"by_phase"`
	} `yaml:"machine"`
}

// TestContractTestSubjectsResolve asserts every contracts/tests/*.test.yaml
// file's subject: field names a path that exists relative to the tree root
// (the directory that has contracts/, roles/, and output-profiles/ as
// siblings). This is the check D9 identified as missing: nothing previously
// validated that subject: pointed at a real file, which is how
// approval-gate.test.yaml's stale subject (contracts/approval-gate.yaml,
// pre-dating the machine/ subdirectory split) survived undetected.
func TestContractTestSubjectsResolve(t *testing.T) {
	t.Parallel()

	roots := []string{
		filepath.Join(repoRoot(t), "internal", "embed", "defaults"),
		isolatedStrategistDir(t),
	}

	for _, root := range roots {
		root := root
		testsDir := filepath.Join(root, "contracts", "tests")
		entries, err := os.ReadDir(testsDir)
		if err != nil {
			t.Fatalf("read %s: %v", testsDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			entry := entry
			testPath := filepath.Join(testsDir, entry.Name())

			t.Run(filepath.Join(filepath.Base(root), entry.Name()), func(t *testing.T) {
				t.Parallel()

				data, err := os.ReadFile(testPath)
				if err != nil {
					t.Fatalf("read %s: %v", testPath, err)
				}

				var fixture contractTestFixture
				if err := yaml.Unmarshal(data, &fixture); err != nil {
					t.Fatalf("parse %s: %v", testPath, err)
				}
				if fixture.Subject == "" {
					t.Fatalf("%s missing required subject: field", testPath)
				}

				subjectPath := filepath.Join(root, filepath.FromSlash(fixture.Subject))
				if _, err := os.Stat(subjectPath); err != nil {
					t.Fatalf("%s declares subject: %s, which does not resolve to an existing file (looked for %s)", testPath, fixture.Subject, subjectPath)
				}
			})
		}
	}
}

// TestContractIndexReferencesAllMachineContracts asserts every contracts/machine/*.yaml
// file is referenced by contracts/index.yaml (machine.always_load or machine.by_phase).
// W2 (02-defects.md D9) wants this enforced; implemented here as a spec test rather than
// a `strategist check` CLI flag: this is skill-corpus self-consistency (shipped content
// authored by Strategist maintainers), not user-workspace configuration — a `check`
// failure would fire for every end user of a stale release, when the actual fix belongs
// at authoring/CI time. It lives next to TestContractTestSubjectsResolve because both
// close the same D9 gap: nothing previously verified the manifest's completeness claim.
func TestContractIndexReferencesAllMachineContracts(t *testing.T) {
	t.Parallel()

	roots := []string{
		filepath.Join(repoRoot(t), "strategist"),
		filepath.Join(repoRoot(t), "internal", "embed", "defaults"),
		isolatedStrategistDir(t),
	}

	for _, root := range roots {
		root := root
		t.Run(filepath.Base(root), func(t *testing.T) {
			t.Parallel()

			indexPath := filepath.Join(root, "contracts", "index.yaml")
			data, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatalf("read %s: %v", indexPath, err)
			}
			var idx contractIndexFixture
			if err := yaml.Unmarshal(data, &idx); err != nil {
				t.Fatalf("parse %s: %v", indexPath, err)
			}

			referenced := make(map[string]bool)
			for _, rel := range idx.Machine.AlwaysLoad {
				referenced[rel] = true
			}
			for _, rels := range idx.Machine.ByPhase {
				for _, rel := range rels {
					referenced[rel] = true
				}
			}

			machineDir := filepath.Join(root, "contracts", "machine")
			entries, err := os.ReadDir(machineDir)
			if err != nil {
				t.Fatalf("read %s: %v", machineDir, err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				rel := "machine/" + entry.Name()
				if !referenced[rel] {
					t.Errorf("%s: %s is not referenced by contracts/index.yaml (always_load or by_phase)", indexPath, rel)
				}
			}
		})
	}
}
