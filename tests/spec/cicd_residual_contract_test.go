//go:build spec

package spec_test

import (
	"os/exec"
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
		"release-verify: ci-lint ci-test docs-governance-gate validate-fixtures vuln-ci release-reproducible-check",
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

func TestStrategistReleaseEvidenceBoundaryIsDocumented(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	runbook := readFile(t, filepath.Join(root, "docs", "runbooks", "strategist-philosophy-and-release-evidence.md"))
	for _, needle := range []string{
		"Source and generated parity",
		"Snapshot release evidence",
		"Published release evidence",
		"must never be described as proof",
		"The executable routine name is `opportunity_attack`",
		"Product version (`v1.0.17`) is distinct from a skill/package contract version",
	} {
		if !strings.Contains(runbook, needle) {
			t.Fatalf("release evidence runbook missing %q", needle)
		}
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
		"ci-lint: lint-status fmt-check mod-check vet build quality-budget-gate",
		"COMPLEXITY_THRESHOLD ?= 7",
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

func TestFmtCheckIsReadOnly(t *testing.T) {
	t.Parallel()

	makefile := readFile(t, filepath.Join(repoRoot(t), "make", "go.mk"))
	start := strings.Index(makefile, "fmt-check:")
	if start < 0 {
		t.Fatal("make/go.mk is missing fmt-check")
	}
	end := strings.Index(makefile[start:], "\n\n")
	if end < 0 {
		t.Fatal("could not isolate fmt-check recipe")
	}
	recipe := makefile[start : start+end]
	if strings.Contains(recipe, "gofmt -w") {
		t.Fatal("fmt-check must not mutate the worktree")
	}
	if !strings.Contains(recipe, "run 'make fmt'") {
		t.Fatal("fmt-check must direct operators to the explicit formatter target")
	}
}

func TestPromptfooUnavailableEndpointIsActionable(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("make", "-s", "eval-promptfoo", "PROMPTFOO_LM_STUDIO_URL=http://127.0.0.1:1")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("eval-promptfoo should fail when the configured endpoint is unavailable")
	}
	message := string(output)
	for _, needle := range []string{
		"no LM Studio (or compatible) server detected",
		"start it, or override PROMPTFOO_LM_STUDIO_URL",
	} {
		if !strings.Contains(message, needle) {
			t.Fatalf("unavailable Promptfoo endpoint output missing %q: %s", needle, message)
		}
	}
	if strings.Contains(message, "fetch failed") || strings.Contains(message, "ECONNREFUSED") {
		t.Fatalf("raw endpoint failure leaked as the primary diagnostic: %s", message)
	}
}

func TestQuickstartExplainsMissionBoundaries(t *testing.T) {
	t.Parallel()

	quickstart := readFile(t, filepath.Join(repoRoot(t), "docs", "onboarding", "quickstart-concepts.md"))
	for _, needle := range []string{
		"Scout",
		"Local policy gate",
		"Strategist Approval Gate",
		"implementation handoffs still require separate coding work",
		"does not silently replace the provider",
	} {
		if !strings.Contains(quickstart, needle) {
			t.Fatalf("quickstart missing operator boundary %q", needle)
		}
	}
}

func TestDocumentationUsesCurrentRuntimePathModel(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	contributing := readFile(t, filepath.Join(root, "CONTRIBUTING.md"))
	architecture := readFile(t, filepath.Join(root, "docs", "architecture", "overview.md"))
	onboarding := readFile(t, filepath.Join(root, "docs", "onboarding", "readme-en.md"))
	adr := readFile(t, filepath.Join(root, "docs", "adr", "0005-slot-write-contracts.md"))

	for _, doc := range []struct {
		name    string
		content string
	}{
		{"CONTRIBUTING.md", contributing},
		{"docs/architecture/overview.md", architecture},
		{"docs/onboarding/readme-en.md", onboarding},
	} {
		for _, forbidden := range []string{"make sync-embed", "../../strategist/SKILL.md", "../../strategist/protocol.md"} {
			if strings.Contains(doc.content, forbidden) {
				t.Fatalf("%s still references retired path/workflow %q", doc.name, forbidden)
			}
		}
	}
	for _, needle := range []string{
		"Go matching `go.mod` (`go 1.27.1`, toolchain `go1.27.1`)",
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

func TestContributorToolchainMatchesGoMod(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	goMod := readFile(t, filepath.Join(root, "go.mod"))
	contributing := readFile(t, filepath.Join(root, "CONTRIBUTING.md"))
	goVersionPattern := regexp.MustCompile(`(?m)^go\s+(\S+)\s*$`)
	goVersion := goVersionPattern.FindStringSubmatch(goMod)
	if len(goVersion) != 2 {
		t.Fatal("go.mod is missing a canonical go version declaration")
	}
	toolchainVersion := goVersion[1]
	toolchainPattern := regexp.MustCompile(`(?m)^toolchain\s+(go\S+)\s*$`)
	if match := toolchainPattern.FindStringSubmatch(goMod); len(match) == 2 {
		toolchainVersion = strings.TrimPrefix(match[1], "go")
	}
	if !strings.Contains(contributing, "toolchain `go"+toolchainVersion+"`") {
		t.Fatalf("CONTRIBUTING.md does not document the canonical go toolchain %q", toolchainVersion)
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
