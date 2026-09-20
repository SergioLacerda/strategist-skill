//go:build spec

package spec_test

import (
	"fmt"
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
	// Counts come from the reviewed policy files themselves, so adding a
	// package consistently to the manifest never breaks this contract.
	root := repoRoot(t)
	gated := countTSVRows(t, filepath.Join(root, "scripts", "coverage-packages.tsv"))
	exempt := countTSVRows(t, filepath.Join(root, "scripts", "coverage-exemptions.tsv"))
	want := fmt.Sprintf("coverage manifest complete: %d packages (%d gated, %d exempt)", gated+exempt, gated, exempt)
	if !strings.Contains(output, want) {
		t.Fatalf("missing deterministic completeness summary %q: %s", want, output)
	}
}

// countTSVRows counts data rows, ignoring blank lines and # comments.
func countTSVRows(t *testing.T, path string) int {
	t.Helper()
	rows := 0
	for _, line := range strings.Split(readFile(t, path), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			rows++
		}
	}
	return rows
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

// writeFixturePolicy writes a manifest and exemptions pair derived from the
// reviewed policy, appending the given extra rows to each.
func writeFixturePolicy(t *testing.T, extraManifest, extraExemptions string) (manifest, exemptions string) {
	t.Helper()
	root := repoRoot(t)
	fixture := t.TempDir()
	manifest = filepath.Join(fixture, "manifest.tsv")
	exemptions = filepath.Join(fixture, "exemptions.tsv")
	m := readFile(t, filepath.Join(root, "scripts", "coverage-packages.tsv")) + extraManifest
	e := readFile(t, filepath.Join(root, "scripts", "coverage-exemptions.tsv")) + extraExemptions
	if err := os.WriteFile(manifest, []byte(m), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exemptions, []byte(e), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest, exemptions
}

func TestCoverageManifestRejectsInvalidPolicyRows(t *testing.T) {
	cases := []struct {
		name, manifestRow, exemptionRow, want string
	}{
		{"duplicate manifest row", "internal/check\t90\tdup fixture\n", "", "COVERAGE_MANIFEST_INVALID duplicate package: internal/check"},
		{"threshold above 100", "internal/fixture-a\t101\tbad threshold\n", "", "COVERAGE_MANIFEST_INVALID invalid threshold for internal/fixture-a: 101"},
		{"non numeric threshold", "internal/fixture-b\tabc\tbad threshold\n", "", "COVERAGE_MANIFEST_INVALID invalid threshold for internal/fixture-b: abc"},
		{"missing reason", "internal/fixture-c\t90\n", "", "COVERAGE_MANIFEST_INVALID row must contain package, numeric threshold, and reason: internal/fixture-c"},
		{"malformed path", "vendor/x\t90\tout of scope\n", "", "COVERAGE_MANIFEST_INVALID out-of-scope package: vendor/x"},
		{"exemption missing owner", "", "internal/fixture-d\n", "COVERAGE_EXEMPTION_INVALID row must contain package, owner, and reason: internal/fixture-d"},
		{"exemption duplicate", "", "internal/authorization\towner\tdup fixture\n", "COVERAGE_EXEMPTION_INVALID duplicate package: internal/authorization"},
		{"manifest and exemption overlap", "internal/authorization\t90\toverlap fixture\n", "", "COVERAGE_EXEMPTION_INVALID package appears in both manifest and exemptions: internal/authorization"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest, exemptions := writeFixturePolicy(t, tc.manifestRow, tc.exemptionRow)
			output, err := runCoverageManifestCheck(t, manifest, exemptions)
			if err == nil {
				t.Fatalf("expected failure, output: %s", output)
			}
			if !strings.Contains(output, tc.want) {
				t.Fatalf("missing diagnostic %q in: %s", tc.want, output)
			}
		})
	}
}

func TestCoverageManifestDiagnosticsAreSortedAndStable(t *testing.T) {
	manifest, exemptions := writeFixturePolicy(t, "internal/zz-stale\t90\tfixture\ninternal/aa-stale\t90\tfixture\n", "")
	first, err := runCoverageManifestCheck(t, manifest, exemptions)
	if err == nil {
		t.Fatalf("expected failure: %s", first)
	}
	second, _ := runCoverageManifestCheck(t, manifest, exemptions)
	if first != second {
		t.Fatalf("diagnostics not stable across runs:\n%s\n---\n%s", first, second)
	}
	if strings.Index(first, "internal/aa-stale") > strings.Index(first, "internal/zz-stale") {
		t.Fatalf("diagnostics not sorted: %s", first)
	}
}

func TestCoverageGateReportsInventoryFailureBeforeMeasuring(t *testing.T) {
	manifest, exemptions := writeFixturePolicy(t, "internal/not-a-package\t90\tfixture stale row\n", "")
	cache := t.TempDir()
	cmd := exec.Command("bash", "scripts/check-coverage-gate.sh", manifest, t.TempDir(), cache, exemptions)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+cache)
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err == nil {
		t.Fatalf("expected inventory failure: %s", output)
	}
	if !strings.Contains(output, "COVERAGE_MANIFEST_STALE") {
		t.Fatalf("inventory diagnostic missing: %s", output)
	}
	if strings.Contains(output, ">=") || strings.Contains(output, "FAIL:") {
		t.Fatalf("inventory failure must be distinct from coverage measurement/threshold output: %s", output)
	}
}

func TestCoverageManifestReportsOmittedInventoryPackage(t *testing.T) {
	root := repoRoot(t)
	fixture := t.TempDir()
	var kept []string
	for _, line := range strings.Split(readFile(t, filepath.Join(root, "scripts", "coverage-packages.tsv")), "\n") {
		if strings.HasPrefix(line, "internal/check\t") {
			continue
		}
		kept = append(kept, line)
	}
	manifest := filepath.Join(fixture, "manifest.tsv")
	if err := os.WriteFile(manifest, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := runCoverageManifestCheck(t, manifest, "scripts/coverage-exemptions.tsv")
	if err == nil {
		t.Fatalf("expected omitted-package failure: %s", output)
	}
	if !strings.Contains(output, "COVERAGE_INVENTORY_OMITTED discovered package has no manifest row or reviewed exemption: internal/check") {
		t.Fatalf("missing omitted diagnostic: %s", output)
	}
}

func TestCoverageManifestReportsDiscoveryFailureNotMassStale(t *testing.T) {
	output, err := runCoverageManifestCheck(t, "scripts/coverage-packages.tsv", "scripts/coverage-exemptions.tsv")
	if err != nil {
		t.Fatalf("baseline must pass: %v\n%s", err, output)
	}
	cmd := exec.Command("bash", "scripts/check-coverage-manifest.sh", "scripts/coverage-packages.tsv", "scripts/coverage-exemptions.tsv", "/proc/nonexistent/gocache")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE=/proc/nonexistent/gocache")
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		t.Fatalf("expected discovery failure: %s", out)
	}
	text := string(out)
	if !strings.Contains(text, "COVERAGE_INVENTORY_DISCOVERY_FAILED unable to discover production Go packages") {
		t.Fatalf("missing discovery-failure diagnostic: %s", text)
	}
	if strings.Contains(text, "COVERAGE_MANIFEST_STALE") || strings.Contains(text, "COVERAGE_EXEMPTION_STALE") {
		t.Fatalf("discovery failure must not be reported as stale rows: %s", text)
	}
}
