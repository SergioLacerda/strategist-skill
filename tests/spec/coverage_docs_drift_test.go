//go:build spec

package spec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureBadgeJSON = `{"schemaVersion": 1, "label": "coverage", "message": "92.8%", "color": "green"}` + "\n"

func writeReadmeFixture(t *testing.T, badge string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "README.md")
	body := "# T\n\n[![Coverage](https://img.shields.io/badge/coverage-" + badge + "-brightgreen)](docs/test-styles.md)\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runScript(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSyncReadmeCoverageBadgeIsIdempotentAndMatchesMeasured(t *testing.T) {
	readme := writeReadmeFixture(t, "10.0%25")
	coverageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-badge.json"), []byte(fixtureBadgeJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func() string {
		if out, err := runScript(t, []string{"README_FILE=" + readme}, "scripts/sync-readme-coverage-badge.sh", coverageDir); err != nil {
			t.Fatalf("sync failed: %v\n%s", err, out)
		}
		return readFile(t, readme)
	}
	first, second := run(), run()
	if first != second {
		t.Fatalf("readme sync not idempotent:\n%s\n---\n%s", first, second)
	}
	if !strings.Contains(first, "badge/coverage-92.8%25-green") {
		t.Fatalf("badge not updated to measured value/color: %s", first)
	}
}

func TestCoverageDocsDriftCheckDetectsStaleReadmeBadge(t *testing.T) {
	coverageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-badge.json"), []byte(fixtureBadgeJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := os.Getenv("GOCACHE")
	if cache == "" {
		cache = t.TempDir()
	}
	docEnv := "TEST_STYLES_DOC=" + filepath.Join(repoRoot(t), "docs", "test-styles.md")

	stale := writeReadmeFixture(t, "50.0%25")
	out, err := runScript(t, []string{"README_FILE=" + stale, docEnv, "GOCACHE=" + cache}, "scripts/check-coverage-docs-drift.sh", coverageDir, cache)
	if err == nil {
		t.Fatalf("expected drift failure: %s", out)
	}
	if !strings.Contains(out, "COVERAGE_DOCS_DRIFT README coverage badge: documented 50.0% vs measured 92.8%") {
		t.Fatalf("missing stable drift diagnostic: %s", out)
	}

	// Profiles measured above are reused (same coverageDir), so this is fast.
	fresh := writeReadmeFixture(t, "92.8%25")
	out, err = runScript(t, []string{"README_FILE=" + fresh, docEnv, "COVERAGE_DOCS_TOLERANCE=5", "GOCACHE=" + cache}, "scripts/check-coverage-docs-drift.sh", coverageDir, cache)
	if err != nil {
		t.Fatalf("in-sync docs must pass: %v\n%s", err, out)
	}
}

func TestDocsGeneratedGateRunsCoverageDocsDriftCheck(t *testing.T) {
	gate := readFile(t, filepath.Join(repoRoot(t), "make", "governance.mk"))
	i := strings.Index(gate, "docs-generated-gate: build")
	j := strings.Index(gate, "contract-consistency-gate:")
	if i < 0 || j < i || !strings.Contains(gate[i:j], "coverage-docs-drift-check") {
		t.Fatal("docs-generated-gate must invoke coverage-docs-drift-check (sync verification)")
	}
}
