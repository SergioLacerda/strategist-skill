//go:build spec

package spec_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseVerificationAndConcurrencyContracts(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	makefile := readMakefileSystem(t, root)
	releaseWorkflow := readFile(t, filepath.Join(root, ".github", "workflows", "release.yml"))

	for _, dep := range []string{
		"release-verify: ci-lint ci-test docs-governance-gate validate-fixtures release-reproducible-check",
		"quality-budget-gate: install-gocognit",
		"release-reproducible-check:",
	} {
		if !strings.Contains(makefile, dep) {
			t.Fatalf("Makefile missing release verification contract %q", dep)
		}
	}
	if strings.Contains(makefile, "release-verify: vet test-lite") {
		t.Fatalf("release-verify must not regress to the lite preflight")
	}
	if !strings.Contains(releaseWorkflow, "group: release-${{ github.repository }}") {
		t.Fatalf("release workflow must serialize releases at repository scope")
	}
	if strings.Contains(releaseWorkflow, "group: release-${{ github.ref }}") {
		t.Fatalf("release workflow must not serialize releases per tag ref")
	}
	if !strings.Contains(releaseWorkflow, "cancel-in-progress: false") {
		t.Fatalf("release workflow must never cancel a release in flight")
	}
}

func TestQualityAndSecurityGateContracts(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	makefile := readMakefileSystem(t, root)
	qualityScript := readFile(t, filepath.Join(root, "scripts", "check-quality-budgets.sh"))
	budgets := readFile(t, filepath.Join(root, "scripts", "quality-budgets.tsv"))
	golangci := readFile(t, filepath.Join(root, ".golangci.yaml"))

	for _, needle := range []string{
		"ci-lint: fmt-check mod-check vet build quality-budget-gate",
		"COMPLEXITY_THRESHOLD ?= 15",
		"GOCOGNIT_VERSION    ?= v1.2.1",
	} {
		if !strings.Contains(makefile, needle) {
			t.Fatalf("Makefile missing quality gate contract %q", needle)
		}
	}
	for _, needle := range []string{
		"gocognit",
		"-over \"$complexity_threshold\"",
		"budget is $limit",
		"stale budget for missing file",
	} {
		if !strings.Contains(qualityScript, needle) {
			t.Fatalf("quality budget script missing %q", needle)
		}
	}
	if !strings.Contains(budgets, "cmd/strategist/dojo.go\t230") {
		t.Fatalf("quality budget manifest must record reviewed file-size baselines")
	}
	if strings.Contains(golangci, "- G304") {
		t.Fatalf("G304 must not be globally excluded in .golangci.yaml")
	}
	genericG304 := regexp.MustCompile(`//nolint:gosec\s*//\s*G304\s*(\n|$)`)
	for _, path := range []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "internal"),
	} {
		if genericG304.MatchString(readTree(t, path)) {
			t.Fatalf("G304 suppressions must carry local review rationale under %s", path)
		}
	}
}

func TestDocumentationUsesCurrentRuntimePathModel(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	contributing := readFile(t, filepath.Join(root, "CONTRIBUTING.md"))
	architecture := readFile(t, filepath.Join(root, "docs", "architecture.md"))
	onboarding := readFile(t, filepath.Join(root, "docs", "onboarding", "readme-en.md"))
	adr := readFile(t, filepath.Join(root, "docs", "adr", "0005-slot-write-contracts.md"))

	for _, doc := range []struct {
		name    string
		content string
	}{
		{"CONTRIBUTING.md", contributing},
		{"docs/architecture.md", architecture},
		{"docs/onboarding/readme-en.md", onboarding},
	} {
		for _, forbidden := range []string{"make sync-embed", "../../strategist/SKILL.md", "../../strategist/protocol.md"} {
			if strings.Contains(doc.content, forbidden) {
				t.Fatalf("%s still references retired path/workflow %q", doc.name, forbidden)
			}
		}
	}
	for _, needle := range []string{
		"Go matching `go.mod` (`go 1.26.4`, toolchain `go1.26.5`)",
		"Node.js 22",
		"`internal/embed/defaults/` is the single authoring source",
	} {
		if !strings.Contains(contributing, needle) {
			t.Fatalf("CONTRIBUTING.md missing current contributor instruction %q", needle)
		}
	}
	for _, needle := range []string{
		"retired root `strategist/` authoring mirror is not a current build, documentation, or runtime source",
		"Runtime defaults are authored in `internal/embed/defaults/`",
	} {
		if !strings.Contains(architecture, needle) {
			t.Fatalf("architecture docs missing current path model %q", needle)
		}
	}
	if !strings.Contains(onboarding, "embedded runtime defaults") {
		t.Fatalf("onboarding should point to embedded runtime defaults")
	}
	for _, needle := range []string{"Declarative", "Detective", "Preventive"} {
		if !strings.Contains(adr, needle) {
			t.Fatalf("ADR-0005 missing enforcement level %q", needle)
		}
	}
}

func TestReproducibleBuildProofContract(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	script := readFile(t, filepath.Join(root, "scripts", "check-reproducible-build.sh"))
	for _, needle := range []string{
		"go build",
		"-trimpath",
		"-X main.Version=reproducible-check",
		"sha256sum",
		"repeated deterministic builds produced different checksums",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("reproducible build script missing %q", needle)
		}
	}
}

func readTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for rel := range relativeFileSet(t, root) {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		b.WriteString(readFile(t, filepath.Join(root, filepath.FromSlash(rel))))
		b.WriteByte('\n')
	}
	return b.String()
}
