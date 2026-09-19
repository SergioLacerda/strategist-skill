//go:build spec

package spec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCoverageManifestCheck(t *testing.T, manifest, exemptions string) (string, error) {
	t.Helper()
	cache := t.TempDir()
	cmd := exec.Command("bash", "scripts/check-coverage-manifest.sh", manifest, exemptions, cache)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+cache)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestCoverageManifestCompletenessAcceptsReviewedInventory(t *testing.T) {
	output, err := runCoverageManifestCheck(t, "scripts/coverage-packages.tsv", "scripts/coverage-exemptions.tsv")
	if err != nil {
		t.Fatalf("coverage manifest check failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "coverage manifest complete: 43 packages (34 gated, 9 exempt)") {
		t.Fatalf("missing deterministic completeness summary: %s", output)
	}
}

func TestCoverageManifestCompletenessReportsStableStaleDiagnostic(t *testing.T) {
	root := repoRoot(t)
	fixture := t.TempDir()
	manifest := filepath.Join(fixture, "manifest.tsv")
	exemptions := filepath.Join(fixture, "exemptions.tsv")
	manifestContent := readFile(t, filepath.Join(root, "scripts", "coverage-packages.tsv"))
	manifestContent += "internal/not-a-package\t90\tfixture stale row\n"
	if err := os.WriteFile(manifest, []byte(manifestContent), 0o644); err != nil {
		t.Fatal(err)
	}
	exemptionContent := readFile(t, filepath.Join(root, "scripts", "coverage-exemptions.tsv"))
	if err := os.WriteFile(exemptions, []byte(exemptionContent), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := runCoverageManifestCheck(t, manifest, exemptions)
	if err == nil {
		t.Fatalf("expected stale manifest failure, output: %s", output)
	}
	if !strings.Contains(output, "COVERAGE_MANIFEST_STALE package is not discovered: internal/not-a-package") {
		t.Fatalf("missing stable stale diagnostic: %s", output)
	}
}
