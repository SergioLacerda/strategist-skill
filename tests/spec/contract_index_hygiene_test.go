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
