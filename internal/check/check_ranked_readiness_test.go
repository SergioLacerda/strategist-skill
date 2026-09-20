package check

import (
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateRankedRuntimeRootRequiresDirectOpenSpecConfig(t *testing.T) {
	runtimeRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(runtimeRoot, "config.yaml"), []byte("schema: spec-driven\n"), 0o644))

	result := validateRankedRuntimeRoot(runtimeRoot, "openspec-propose")

	require.True(t, result.Ready())
}

func TestValidateRankedRuntimeRootRejectsNestedOpenSpecConfig(t *testing.T) {
	runtimeRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(runtimeRoot, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeRoot, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))

	result := validateRankedRuntimeRoot(runtimeRoot, "openspec-propose")

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_missing", result.ReasonCode)
}

func TestValidateRankedRuntimeContractRejectsInvalidRuntime(t *testing.T) {
	result := validateRankedRuntimeContract(domain.RankedRuntimeContract{Kind: "unsupported"}, "refinement", "provider")

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_contract_invalid", result.ReasonCode)
	require.Contains(t, result.Detail, "role/provider=refinement/provider")
}

func TestValidateRankedRuntimeContractAllowsNoRuntime(t *testing.T) {
	result := validateRankedRuntimeContract(domain.RankedRuntimeContract{Kind: domain.RankedRuntimeNone}, "discovery", "provider")

	require.Equal(t, domain.ReadinessReady, result.Status)
	require.Equal(t, "ranked_runtime_not_required", result.ReasonCode)
}

func TestReadRankedRuntimeStateReportsMissingAndInvalidFiles(t *testing.T) {
	root := t.TempDir()
	missing, result := readRankedRuntimeState(root, "provider", ".strategist/runtime")
	require.Empty(t, missing.Entries)
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_state_missing", result.ReasonCode)

	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte("not-json"), 0o644))
	_, result = readRankedRuntimeState(root, "provider", ".strategist/runtime")
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_state_invalid", result.ReasonCode)
}

func TestMatchRankedRuntimeStateReportsMismatchAndMissingBinding(t *testing.T) {
	state := rankedRuntimeStateCheck{Entries: []rankedRuntimeStateEntryCheck{{Slot: "refinement", Provider: "provider", ContractDigest: "sha256:observed"}}}

	mismatch := matchRankedRuntimeState(state, "refinement", "provider", "sha256:expected", ".strategist/runtime")
	require.Equal(t, "ranked_runtime_digest_mismatch", mismatch.ReasonCode)

	missing := matchRankedRuntimeState(state, "discovery", "provider", "sha256:expected", ".strategist/runtime")
	require.Equal(t, "ranked_runtime_binding_missing", missing.ReasonCode)

	ready := matchRankedRuntimeState(state, "refinement", "provider", "sha256:observed", ".strategist/runtime")
	require.True(t, ready.Ready())
}

func TestRankedRuntimeReadinessReportsMissingRoot(t *testing.T) {
	root := t.TempDir()
	const digest = "sha256:certified"
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{"entries":[{"slot":"refinement","provider":"provider","contract_digest":"`+digest+`"}]}`), 0o644))

	result := rankedRuntimeReadiness(root, "refinement", "provider", domain.CatalogRankedStamp{
		CertificationDigest: digest,
		Runtime: domain.RankedRuntimeContract{
			Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/runtime", Bootstrap: "bootstrap", Healthcheck: "healthcheck",
		},
	})

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_missing", result.ReasonCode)
}

func TestRankedRuntimeReadinessReportsRuntimeNotRequired(t *testing.T) {
	result := rankedRuntimeReadiness(t.TempDir(), "discovery", "brainstorming", domain.CatalogRankedStamp{
		Runtime: domain.RankedRuntimeContract{Kind: domain.RankedRuntimeNone},
	})

	require.Equal(t, domain.ReadinessReady, result.Status)
	require.Equal(t, "ranked_runtime_not_required", result.ReasonCode)
}

func TestLiveHostAPIDigestFallsBackAndComputesMaterializedContract(t *testing.T) {
	root := t.TempDir()
	const fallback = "sha256:fallback"

	require.Equal(t, fallback, liveHostAPIDigest(root, "discovery", fallback))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "default.yaml"), []byte("discovery: ranger\n"), 0o644))
	require.Equal(t, fallback, liveHostAPIDigest(root, "discovery", fallback))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "ranger.yaml"), []byte("role: ranger\nslot: discovery\n"), 0o644))
	require.Equal(t, fallback, liveHostAPIDigest(root, "discovery", fallback))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal_skills", "ranger"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "internal_skills", "ranger", "SKILL.md"), []byte("ranger"), 0o644))
	computed := liveHostAPIDigest(root, "discovery", fallback)
	require.True(t, strings.HasPrefix(computed, "sha256:"))
	require.NotEqual(t, fallback, computed)
}

func TestRankedCertificationReadinessReportsInvalidCatalogAndMissingRuntimeState(t *testing.T) {
	root := t.TempDir()
	plugins := filepath.Join(root, "plugins")
	require.NoError(t, os.MkdirAll(plugins, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(plugins, "catalog.yaml"), []byte(": invalid: yaml\n"), 0o644))
	trust, grant := rankedCertificationReadiness(root, "refinement", "provider")
	require.Equal(t, "ranked_catalog_invalid", trust.ReasonCode)
	require.Equal(t, trust.ReasonCode, grant.ReasonCode)

	const digest = "sha256:certified"
	require.NoError(t, os.WriteFile(filepath.Join(plugins, "catalog.yaml"), []byte(`schema_version: strategist-plugin-catalog/v1
providers:
  - id: provider
    ranked: true
    certification_digest: `+digest+`
    host_api_digest: sha256:host
    connector_digest: sha256:connector
    test_suite_digest: sha256:tests
    policy_digest: sha256:policy
    conformance_level: C1
    runtime:
      kind: openspec_root
      root: .strategist/runtime
      bootstrap: bootstrap
      healthcheck: healthcheck
`), 0o644))
	trust, grant = rankedCertificationReadiness(root, "refinement", "provider")
	require.Equal(t, "ranked_runtime_state_missing", trust.ReasonCode)
	require.Equal(t, trust.ReasonCode, grant.ReasonCode)
}

func TestRunRankedRuntimeHealthcheckReportsFailureAndSuccess(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	runtimeRoot := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	semanticRoot := filepath.Dir(runtimeRoot)
	script := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o755))
	t.Setenv("PATH", filepath.Dir(script)+string(os.PathListSeparator)+os.Getenv("PATH"))

	failed := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")
	require.Equal(t, domain.ReadinessBlocked, failed.Status)
	require.Equal(t, "ranked_runtime_healthcheck_failed", failed.ReasonCode)

	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"root\":{\"path\":\""+semanticRoot+"\"}}'"), 0o755))
	passed := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")
	require.Equal(t, domain.ReadinessReady, passed.Status)
	require.Equal(t, "ranked_runtime_healthy", passed.ReasonCode)
	require.Contains(t, passed.Detail, runtimeRoot)
}

func TestRunRankedRuntimeHealthcheckRejectsSemanticRootMismatch(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	runtimeRoot := t.TempDir()
	script := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"root\":{\"path\":\"/wrong/root\"}}'"), 0o755))
	t.Setenv("PATH", filepath.Dir(script)+string(os.PathListSeparator)+os.Getenv("PATH"))

	result := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_mismatch", result.ReasonCode)
	require.Contains(t, result.Detail, "expected")
}

// Contract test against the real openspec binary so stubs cannot mask its
// canonical (absolute, symlink-resolved) root.path output.
func TestRunRankedRuntimeHealthcheckRealOpenSpecPathForms(t *testing.T) {
	if _, err := exec.LookPath("openspec"); err != nil {
		t.Skip("openspec binary not available")
	}
	base := t.TempDir()
	realRoot := filepath.Join(base, "real")
	runtimeRoot := filepath.Join(realRoot, ".strategist", "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeRoot, "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(realRoot, link))
	t.Chdir(realRoot)

	for name, root := range map[string]string{
		"absolute": runtimeRoot,
		"relative": filepath.Join(".strategist", "openspec"),
		"symlink":  filepath.Join(link, ".strategist", "openspec"),
	} {
		t.Run(name, func(t *testing.T) {
			result := runRankedRuntimeHealthcheck(root, "openspec-propose")
			require.Equal(t, domain.ReadinessReady, result.Status, result.Detail)
		})
	}
}

// Drift regression: the provider reports an absolute root.path while the
// declared runtime root is relative. A fake provider keeps this covered on
// hosts without an openspec binary.
func TestRunRankedRuntimeHealthcheckAcceptsRelativeRootWithAbsoluteReport(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	base := t.TempDir()
	semanticRoot := filepath.Join(base, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(semanticRoot, "openspec"), 0o755))
	script := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '{\"root\":{\"path\":\""+semanticRoot+"\"}}'"), 0o755))
	t.Setenv("PATH", filepath.Dir(script)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(base)

	relative := runRankedRuntimeHealthcheck(filepath.Join(".strategist", "openspec"), "openspec-propose")
	require.Equal(t, domain.ReadinessReady, relative.Status, relative.Detail)

	require.NoError(t, os.MkdirAll(filepath.Join(base, "other", "openspec"), 0o755))
	wrong := runRankedRuntimeHealthcheck(filepath.Join("other", "openspec"), "openspec-propose")
	require.Equal(t, domain.ReadinessBlocked, wrong.Status)
	require.Equal(t, "ranked_runtime_root_mismatch", wrong.ReasonCode)
}

func TestRunRankedRuntimeHealthcheckReportsMissingExecutable(t *testing.T) {
	runtimeRoot := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	t.Setenv("PATH", t.TempDir())

	got := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, got.ReasonCode)
	require.Contains(t, got.Detail, `"openspec-propose"`)
	require.Contains(t, got.Detail, "standalone-runtime-hermeticity.md")
}

func writePrivateNode(t *testing.T, strategist string) rankedRuntimeStatePrivate {
	t.Helper()
	dir := filepath.Join(strategist, "weapon-runtime", "openspec-propose", "node", "bin")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	body := "#!/bin/sh\nprintf '{\"root\":{\"path\":\"%s\"}}' \"${PWD%/*}\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node"), []byte(body), 0o755))
	return rankedRuntimeStatePrivate{
		Node:   "weapon-runtime/openspec-propose/node/bin/node",
		Script: "weapon-runtime/openspec-propose/openspec/openspec.js",
	}
}

func TestPrivateRankedRuntimeHealthcheckRunsWithoutHostOpenSpec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Node is a POSIX shell script")
	}
	strategist := t.TempDir()
	runtimeRoot := filepath.Join(strategist, "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	private := writePrivateNode(t, strategist)
	t.Setenv("PATH", t.TempDir())

	got := runPrivateRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", private)

	require.Equal(t, domain.ReadinessReady, got.Status, got.Detail)
	require.Equal(t, "ranked_runtime_healthy", got.ReasonCode)
}

func TestPrivateRankedRuntimeHealthcheckReportsMissingRuntimeAndEscapes(t *testing.T) {
	strategist := t.TempDir()
	runtimeRoot := filepath.Join(strategist, "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))

	missing := runPrivateRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose",
		rankedRuntimeStatePrivate{Node: "weapon-runtime/openspec-propose/node/bin/node", Script: "weapon-runtime/openspec-propose/openspec/openspec.js"})
	require.Equal(t, domain.ReadinessBlocked, missing.Status)
	require.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, missing.ReasonCode)

	for _, bad := range []rankedRuntimeStatePrivate{
		{Node: "../outside/node", Script: "weapon-runtime/x.js"},
		{Node: "/abs/node", Script: "weapon-runtime/x.js"},
		{Node: "openspec/node", Script: "weapon-runtime/x.js"},
	} {
		escaped := runPrivateRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", bad)
		require.Equal(t, domain.ReadinessBlocked, escaped.Status, bad.Node)
		require.Equal(t, "ranked_runtime_state_invalid", escaped.ReasonCode, bad.Node)
	}
}
