package check

import (
	"os"
	"path/filepath"
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
	state := rankedRuntimeStateCheck{Entries: []struct {
		Slot           string `json:"slot"`
		Provider       string `json:"provider"`
		ContractDigest string `json:"contract_digest"`
	}{{Slot: "refinement", Provider: "provider", ContractDigest: "sha256:observed"}}}

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
	runtimeRoot := t.TempDir()
	script := filepath.Join(t.TempDir(), "openspec")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o755))
	t.Setenv("PATH", filepath.Dir(script)+string(os.PathListSeparator)+os.Getenv("PATH"))

	failed := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")
	require.Equal(t, domain.ReadinessBlocked, failed.Status)
	require.Equal(t, "ranked_runtime_healthcheck_failed", failed.ReasonCode)

	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '{}'"), 0o755))
	passed := runRankedRuntimeHealthcheck(runtimeRoot, "openspec-propose")
	require.Equal(t, domain.ReadinessReady, passed.Status)
	require.Equal(t, "ranked_runtime_healthy", passed.ReasonCode)
	require.Contains(t, passed.Detail, runtimeRoot)
}
