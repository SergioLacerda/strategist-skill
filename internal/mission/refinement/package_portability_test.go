package refinement

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableRuntimeRefIsWorkspaceRelative(t *testing.T) {
	project := t.TempDir()
	got := portableRuntimeRef(OpenSpecInput{BasePath: filepath.Join(project, ".analysis"), RuntimeRoot: filepath.Join(project, ".strategist", "openspec")})
	if got != ".strategist/openspec" {
		t.Fatalf("portableRuntimeRef = %q", got)
	}
}

func TestCheckPackagePortabilityReportsAbsoluteRuntime(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nprovider_runtime: /old/workspace/.strategist/openspec\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckPackagePortability(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Reason != "absolute provider_runtime" {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}
