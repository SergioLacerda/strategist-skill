package check

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"

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

func TestReadRankedRuntimeStateReportsMissingInvalidAndLegacyFiles(t *testing.T) {
	root := t.TempDir()
	missing, result := readRankedRuntimeState(root, "provider", ".strategist/runtime")
	require.Empty(t, missing.Entries)
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_state_missing", result.ReasonCode)

	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte("not-json"), 0o644))
	_, result = readRankedRuntimeState(root, "provider", ".strategist/runtime")
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, domain.ReasonRankedRuntimeStateInvalid, result.ReasonCode)

	legacy := `{"schema_version":"strategist-ranked-runtime/v1","entries":[{"slot":"refinement","provider":"provider","runtime":{"mode":"payload","node":"weapon-runtime/provider/node","script":"weapon-runtime/provider/openspec/x.mjs"}}]}`
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(legacy), 0o644))
	_, result = readRankedRuntimeState(root, "provider", ".strategist/runtime")
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, domain.ReasonRankedRuntimeStateLegacy, result.ReasonCode)
	require.Contains(t, result.Detail, "strategist upgrade")
}

func TestMatchRankedRuntimeStateReportsMismatchAndMissingBinding(t *testing.T) {
	state := domain.RankedRuntimeState{Entries: []domain.RankedRuntimeStateEntry{{Role: "archivist", Slot: "refinement", Provider: "provider", ContractDigest: "sha256:observed"}}}

	_, mismatch := matchRankedRuntimeState(state, "refinement", "provider", "sha256:expected", "archivist", ".strategist/runtime")
	require.Equal(t, "ranked_runtime_digest_mismatch", mismatch.ReasonCode)

	_, missing := matchRankedRuntimeState(state, "discovery", "provider", "sha256:expected", "archivist", ".strategist/runtime")
	require.Equal(t, "ranked_runtime_binding_missing", missing.ReasonCode)

	entry, ready := matchRankedRuntimeState(state, "refinement", "provider", "sha256:observed", "archivist", ".strategist/runtime")
	require.True(t, ready.Ready())
	require.Equal(t, "archivist", entry.Role)

	_, unknownRole := matchRankedRuntimeState(state, "refinement", "provider", "sha256:observed", "", ".strategist/runtime")
	require.True(t, unknownRole.Ready(), "an unreadable role map is reported by role compatibility, not here")
}

// The runtime was recorded for another role than the one the slot now maps to:
// plugins.lock and ranked-runtimes.yaml no longer describe the same binding.
func TestMatchRankedRuntimeStateRejectsARoleMismatch(t *testing.T) {
	state := domain.RankedRuntimeState{Entries: []domain.RankedRuntimeStateEntry{{Role: "ranger", Slot: "refinement", Provider: "provider", ContractDigest: "sha256:d"}}}

	_, got := matchRankedRuntimeState(state, "refinement", "provider", "sha256:d", "archivist", ".strategist/runtime")

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, "ranked_runtime_binding_missing", got.ReasonCode)
	require.Contains(t, got.Detail, "expected role=archivist")
	require.Contains(t, got.Detail, "observed role=ranger")
}

func TestRankedRuntimeReadinessReportsMissingRoot(t *testing.T) {
	root := t.TempDir()
	const digest = "sha256:certified"
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{"schema_version":"`+domain.RankedRuntimeStateSchemaVersion+`","entries":[{"slot":"refinement","provider":"provider","contract_digest":"`+digest+`"}]}`), 0o644))

	result := rankedRuntimeReadiness(root, "refinement", "provider", domain.CatalogRankedStamp{
		CertificationDigest: digest,
		Runtime: domain.RankedRuntimeContract{
			Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/runtime", Bootstrap: "bootstrap", Healthcheck: "healthcheck",
		},
	})

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_missing", result.ReasonCode)
}

func TestRankedRuntimeReadinessReportsAnEntryWithoutRuntime(t *testing.T) {
	root := t.TempDir()
	const digest = "sha256:certified"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "runtime"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "runtime", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(`{"schema_version":"`+domain.RankedRuntimeStateSchemaVersion+`","entries":[{"slot":"refinement","provider":"provider","contract_digest":"`+digest+`"}]}`), 0o644))

	result := rankedRuntimeReadiness(root, "refinement", "provider", domain.CatalogRankedStamp{
		CertificationDigest: digest,
		Runtime:             domain.RankedRuntimeContract{Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/runtime", Bootstrap: "bootstrap", Healthcheck: "healthcheck"},
	})

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, domain.ReasonRankedRuntimeStateInvalid, result.ReasonCode)
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

// A runtime fault is reported as the dependencies (runtime) dimension; trust and
// permission grants stay with the certification that actually decides them.
func TestRankedCertificationReadinessReportsInvalidCatalogAndMissingRuntimeState(t *testing.T) {
	root := t.TempDir()
	plugins := filepath.Join(root, "plugins")
	require.NoError(t, os.MkdirAll(plugins, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(plugins, "catalog.yaml"), []byte(": invalid: yaml\n"), 0o644))
	trust, grant, runtimeCheck := rankedCertificationReadiness(root, "refinement", "provider")
	require.Equal(t, "ranked_catalog_invalid", trust.ReasonCode)
	require.Equal(t, trust.ReasonCode, grant.ReasonCode)
	require.Equal(t, domain.ReadinessUnknown, runtimeCheck.Status, "no runtime verdict without a certified provider")

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
	trust, grant, runtimeCheck = rankedCertificationReadiness(root, "refinement", "provider")
	require.NotEqual(t, "ranked_runtime_state_missing", trust.ReasonCode)
	require.NotEqual(t, "ranked_runtime_state_missing", grant.ReasonCode)
	require.Equal(t, domain.ReadinessBlocked, runtimeCheck.Status)
	require.Equal(t, "ranked_runtime_state_missing", runtimeCheck.ReasonCode)
}

func TestBlockedRuntimeIsReportedAsTheDependenciesDimension(t *testing.T) {
	blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeStateLegacy, Detail: "x"}
	ready := domain.ReadinessCheck{Status: domain.ReadinessReady}

	errs := blockedReadinessErrors("refinement", skillProviderVector("p", "provider", ready, ready, ready, blocked, connectors.ConnectorResult{}, connectors.ObservationResult{}))

	var runtimeErrs []string
	for _, e := range errs {
		if strings.Contains(e, domain.ReasonRankedRuntimeStateLegacy) {
			runtimeErrs = append(runtimeErrs, e)
		}
	}
	require.Len(t, runtimeErrs, 1)
	require.Contains(t, runtimeErrs[0], "readiness blocked on dependencies dimension")
	require.NotContains(t, runtimeErrs[0], "trust")
	require.NotContains(t, runtimeErrs[0], "permission_grant")
}

// fakeHostNode is a POSIX script standing in for the host Node: it reports
// version on --version and otherwise runs body (the healthcheck answer).
func fakeHostNode(version, body string) string {
	return "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then printf 'v" + version + "\\n'; exit 0; fi\n" + body + "\n"
}

// correctlyRootedAnswer answers the healthcheck the way a correctly rooted
// OpenSpec bundle does: root.path is the parent of the runtime root (cwd).
const correctlyRootedAnswer = `printf '{"root":{"path":"%s"}}' "${PWD%/*}"`

// reportingRoot answers with a fixed root.path, so a test can assert that a
// runtime root pointing somewhere else is rejected.
func reportingRoot(semanticRoot string) string {
	return `printf '{"root":{"path":"` + semanticRoot + `"}}'`
}

// absoluteTool resolves a host tool for use inside a fake Node script. A
// runtime subprocess gets an empty PATH, so such a script cannot call even
// `sleep` by name.
func absoluteTool(t *testing.T, tool string) string {
	t.Helper()
	target, err := exec.LookPath(tool)
	if err != nil {
		t.Skipf("%s not available: %v", tool, err)
	}
	return target
}

// writeHostRuntime installs a fake host Node running nodeBody, a one-file
// OpenSpec bundle under weapon-runtime/openspec-propose/openspec/, and returns
// the runtime record install would write for them, digest included.
func writeHostRuntime(t *testing.T, strategist, nodeBody string) domain.RankedRuntimeStateRuntime {
	t.Helper()
	node := filepath.Join(strategist, "host", "node")
	require.NoError(t, os.MkdirAll(filepath.Dir(node), 0o755))
	require.NoError(t, os.WriteFile(node, []byte(nodeBody), 0o755))
	runtimeDir := filepath.Join(strategist, "weapon-runtime", "openspec-propose")
	require.NoError(t, os.MkdirAll(filepath.Join(runtimeDir, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, "openspec", "openspec.mjs"), []byte("// bundle\n"), 0o644))
	digest, _, err := runtimepayload.TreeDigest(os.DirFS(runtimeDir), "openspec")
	require.NoError(t, err)
	return domain.RankedRuntimeStateRuntime{
		Node:       node,
		Script:     "weapon-runtime/openspec-propose/openspec/openspec.mjs",
		Components: []domain.RankedRuntimeStateComponent{{Name: domain.RankedRuntimeComponentOpenSpec, Version: "1.13.0", SHA256: digest}},
	}
}

// hostRuntimeFixture prepares a Strategist root with its runtime root and a
// host runtime answering with nodeBody.
func hostRuntimeFixture(t *testing.T, nodeBody string) (strategist, runtimeRoot string, state domain.RankedRuntimeStateRuntime) {
	t.Helper()
	testutil.RequirePOSIXShell(t)
	strategist = t.TempDir()
	runtimeRoot = filepath.Join(strategist, "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	t.Setenv("PATH", t.TempDir())
	return strategist, runtimeRoot, writeHostRuntime(t, strategist, nodeBody)
}

// hostNodeContract is the certified runtime contract the host Node is checked
// against.
func hostNodeContract() domain.RankedRuntimeContract {
	return domain.RankedRuntimeContract{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec",
		Bootstrap: "openspec init", Healthcheck: "openspec context --json",
		Version: "1.13.0", NodeVersion: "22.23.2",
	}
}

func TestHostNodeRankedRuntimeHealthcheckReportsFailureAndSuccess(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", "exit 1"))

	failed := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessBlocked, failed.Status)
	require.Equal(t, "ranked_runtime_healthcheck_failed", failed.ReasonCode)

	require.NoError(t, os.WriteFile(state.Node, []byte(fakeHostNode("22.23.2", correctlyRootedAnswer)), 0o755))
	passed := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessReady, passed.Status, passed.Detail)
	require.Equal(t, "ranked_runtime_healthy", passed.ReasonCode)
	require.Contains(t, passed.Detail, runtimeRoot)
}

func TestHostNodeRankedRuntimeHealthcheckRejectsSemanticRootMismatch(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", reportingRoot("/wrong/root")))

	result := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_mismatch", result.ReasonCode)
	require.Contains(t, result.Detail, "expected")
}

// Drift regression: the provider reports an absolute root.path while the
// declared runtime root is relative.
func TestHostNodeRankedRuntimeHealthcheckAcceptsRelativeRootWithAbsoluteReport(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	base := t.TempDir()
	strategist := filepath.Join(base, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(strategist, "openspec"), 0o755))
	state := writeHostRuntime(t, strategist, fakeHostNode("22.23.2", reportingRoot(strategist)))
	t.Setenv("PATH", t.TempDir())
	t.Chdir(base)

	relative := runHostNodeRankedRuntimeHealthcheck(strategist, filepath.Join(".strategist", "openspec"), "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessReady, relative.Status, relative.Detail)

	require.NoError(t, os.MkdirAll(filepath.Join(base, "other", "openspec"), 0o755))
	wrong := runHostNodeRankedRuntimeHealthcheck(strategist, filepath.Join("other", "openspec"), "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessBlocked, wrong.Status)
	require.Equal(t, "ranked_runtime_root_mismatch", wrong.ReasonCode)
}

// Contract test against the real embedded OpenSpec bundle run by the real host
// Node, so stubs cannot mask OpenSpec's canonical (absolute, symlink-resolved)
// root.path output. OpenSpec itself is never resolved from PATH.
func TestHostNodeRankedRuntimeHealthcheckRealOpenSpecPathForms(t *testing.T) {
	repo, strategist, state := materializedHostRuntime(t)
	runtimeRoot := filepath.Join(strategist, "openspec")
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(repo, link))
	t.Chdir(repo)

	for name, root := range map[string]string{
		"absolute": runtimeRoot,
		"relative": filepath.Join(".strategist", "openspec"),
		"symlink":  filepath.Join(link, ".strategist", "openspec"),
	} {
		t.Run(name, func(t *testing.T) {
			result := runHostNodeRankedRuntimeHealthcheck(strategist, root, "openspec-propose", hostNodeContract(), state)
			require.True(t, result.Ready(), result.Detail)
		})
	}
}

// materializedHostRuntime materializes the embedded OpenSpec bundle into a
// temporary workspace and pairs it with the host's own Node. It skips when no
// supported Node is installed.
func materializedHostRuntime(t *testing.T) (repo, strategist string, state domain.RankedRuntimeStateRuntime) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinked workspace fixture is POSIX-only")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("host Node not available: %v", err)
	}
	out, err := exec.Command(node, "--version").Output() //nolint:gosec // test-only probe of the host Node
	if err != nil || !domain.VersionAtLeast(domain.ParseReportedVersion(out), domain.MinimumOpenSpecNodeVersion) {
		t.Skipf("host Node %s does not satisfy >=%s", strings.TrimSpace(string(out)), domain.MinimumOpenSpecNodeVersion)
	}
	repo = t.TempDir()
	strategist = filepath.Join(repo, ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(strategist, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategist, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))
	script, evidence, err := runtimepayload.MaterializeOpenSpec(embed.DefaultsFS(), filepath.Join(strategist, "weapon-runtime", "openspec-propose"))
	require.NoError(t, err)
	rel, err := filepath.Rel(strategist, script)
	require.NoError(t, err)
	absNode, err := filepath.Abs(node)
	require.NoError(t, err)
	state = domain.RankedRuntimeStateRuntime{Node: absNode, Script: filepath.ToSlash(rel)}
	for _, c := range evidence.Components {
		state.Components = append(state.Components, domain.RankedRuntimeStateComponent{Name: c.Name, Version: c.Version, SHA256: c.SHA256})
	}
	return repo, strategist, state
}

func TestHostNodeRankedRuntimeHealthcheckReportsMissingExecutable(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, "")
	state.Node = filepath.Join(t.TempDir(), "node")

	got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, got.ReasonCode)
	require.Contains(t, got.Detail, `"openspec-propose"`)
	require.Contains(t, got.Detail, "standalone-runtime-hermeticity.md")
}

func TestHostNodeRankedRuntimeHealthcheckRejectsUnsafeRecordedPaths(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", correctlyRootedAnswer))

	for name, mutate := range map[string]func(*domain.RankedRuntimeStateRuntime){
		"relative node": func(s *domain.RankedRuntimeStateRuntime) { s.Node = "host/node" },
		"escaping script": func(s *domain.RankedRuntimeStateRuntime) {
			s.Script = "weapon-runtime/openspec-propose/openspec/../../../x.mjs"
		},
		"other provider":       func(s *domain.RankedRuntimeStateRuntime) { s.Script = "weapon-runtime/other/openspec/openspec.mjs" },
		"outside openspec dir": func(s *domain.RankedRuntimeStateRuntime) { s.Script = "weapon-runtime/openspec-propose/x.mjs" },
		"no bundle digest":     func(s *domain.RankedRuntimeStateRuntime) { s.Components = nil },
	} {
		t.Run(name, func(t *testing.T) {
			bad := state
			mutate(&bad)

			got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), bad)

			require.Equal(t, domain.ReadinessBlocked, got.Status)
			require.Equal(t, domain.ReasonRankedRuntimeStateInvalid, got.ReasonCode)
		})
	}
}

// An altered bundle is never executed: the fake Node would answer healthy, so
// only the digest check can block it.
func TestHostNodeRankedRuntimeHealthcheckRejectsATamperedBundle(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", correctlyRootedAnswer))
	require.NoError(t, os.WriteFile(filepath.Join(strategist, filepath.FromSlash(state.Script)), []byte("// altered\n"), 0o644))

	got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeBundleAltered, got.ReasonCode)
	require.Contains(t, got.Detail, "strategist upgrade")
}

func TestHostNodeRankedRuntimeHealthcheckReportsAMissingBundle(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", correctlyRootedAnswer))
	require.NoError(t, os.RemoveAll(filepath.Join(strategist, "weapon-runtime")))

	got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeBundleMissing, got.ReasonCode)
}

func TestMatchRankedRuntimeStateDigestMismatchNamesTheRemedy(t *testing.T) {
	state := domain.RankedRuntimeState{Entries: []domain.RankedRuntimeStateEntry{{Slot: "refinement", Provider: "openspec-propose", ContractDigest: "sha256:old"}}}

	_, got := matchRankedRuntimeState(state, "refinement", "openspec-propose", "sha256:new", "", ".strategist/openspec")

	require.Equal(t, "ranked_runtime_digest_mismatch", got.ReasonCode)
	require.Contains(t, got.Detail, "strategist upgrade")
	require.Contains(t, got.Detail, "sha256:old")
	require.Contains(t, got.Detail, "sha256:new")
}

func writeRuntimeCatalog(t *testing.T, root string, withRuntime bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	runtimeBlock := ""
	if withRuntime {
		runtimeBlock = "    runtime:\n      kind: openspec_root\n      root: .strategist/openspec\n      bootstrap: openspec init --profile core\n      healthcheck: openspec context --json\n"
	}
	body := "schema_version: strategist-plugin-catalog/v1\nproviders:\n  - id: provider\n    ranked: true\n    certification_digest: sha256:certified\n" + runtimeBlock
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(body), 0o644))
}

// A provider that declares a runtime must not read as ready in custom mode when
// there is no executable anywhere: that is the green-but-unusable workspace.
func TestCustomRuntimeReadinessBlocksWhenNoExecutableAndNoPrivateRuntime(t *testing.T) {
	root := t.TempDir()
	writeRuntimeCatalog(t, root, true)
	t.Setenv("PATH", t.TempDir())

	got := customRuntimeReadiness(root, "refinement", "provider")

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeExecutableMissing, got.ReasonCode)
	require.Contains(t, got.Detail, `"provider"`)
	require.Contains(t, got.Detail, "strategist install --wizard")
}

func TestCustomRuntimeReadinessIsUnchangedWhenTheProviderNeedsNoRuntimeOrTheCatalogIsAbsent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	require.Equal(t, domain.ReadinessUnknown, customRuntimeReadiness(root, "refinement", "provider").Status, "no catalog: nothing is known to be required")

	writeRuntimeCatalog(t, root, false)
	require.Equal(t, domain.ReadinessUnknown, customRuntimeReadiness(root, "refinement", "provider").Status)
	require.Equal(t, domain.ReadinessUnknown, customRuntimeReadiness(root, "refinement", "someone-else").Status)
}

func TestCustomRuntimeReadinessAcceptsAHostExecutableOrARecordedRuntime(t *testing.T) {
	testutil.RequirePOSIXShell(t)
	root := t.TempDir()
	writeRuntimeCatalog(t, root, true)

	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "openspec"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", bin)
	require.Equal(t, domain.ReadinessReady, customRuntimeReadiness(root, "refinement", "provider").Status, "host executable")

	t.Setenv("PATH", t.TempDir())
	node := filepath.Join(root, "host", "node")
	require.NoError(t, os.MkdirAll(filepath.Dir(node), 0o755))
	require.NoError(t, os.WriteFile(node, []byte("x"), 0o755))
	script := filepath.Join(root, "weapon-runtime", "provider", "openspec", "x.mjs")
	require.NoError(t, os.MkdirAll(filepath.Dir(script), 0o755))
	require.NoError(t, os.WriteFile(script, []byte("x"), 0o644))
	state := domain.RankedRuntimeState{SchemaVersion: domain.RankedRuntimeStateSchemaVersion, Entries: []domain.RankedRuntimeStateEntry{{
		Slot: "refinement", Provider: "provider",
		Runtime: &domain.RankedRuntimeStateRuntime{Node: node, Script: "weapon-runtime/provider/openspec/x.mjs"},
	}}}
	raw, err := json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), raw, 0o644))
	require.Equal(t, domain.ReadinessReady, customRuntimeReadiness(root, "refinement", "provider").Status, "recorded runtime")

	legacy := `{"schema_version":"strategist-ranked-runtime/v1","entries":[{"slot":"refinement","provider":"provider","runtime":{"node":"` + node + `","script":"weapon-runtime/provider/openspec/x.mjs"}}]}`
	require.NoError(t, os.WriteFile(filepath.Join(root, "ranked-runtimes.yaml"), []byte(legacy), 0o644))
	require.Equal(t, domain.ReadinessBlocked, customRuntimeReadiness(root, "refinement", "provider").Status, "a legacy record is not a usable runtime")
}

func TestRankedHealthcheckTimeoutIsConfigurableAndBounded(t *testing.T) {
	t.Setenv(rankedHealthcheckTimeoutEnv, "")
	require.Equal(t, defaultRankedHealthcheckTimeout, rankedHealthcheckTimeout())
	require.GreaterOrEqual(t, defaultRankedHealthcheckTimeout.Seconds(), 30.0, "a cold first launch must not race the limit")

	t.Setenv(rankedHealthcheckTimeoutEnv, "45s")
	require.Equal(t, 45*time.Second, rankedHealthcheckTimeout())

	for _, bad := range []string{"soon", "-5s", "0s", "24h"} {
		t.Setenv(rankedHealthcheckTimeoutEnv, bad)
		require.Equal(t, defaultRankedHealthcheckTimeout, rankedHealthcheckTimeout(), "invalid or out-of-range %q falls back", bad)
	}
}

func TestRankedHealthcheckReportsATimeoutDistinctFromAFailure(t *testing.T) {
	sleep := absoluteTool(t, "sleep")
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("22.23.2", sleep+" 5"))
	t.Setenv(rankedHealthcheckTimeoutEnv, "300ms")

	got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)

	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, domain.ReasonRankedRuntimeHealthcheckTimeout, got.ReasonCode)
	require.Contains(t, got.Detail, "300ms")
	require.Contains(t, got.Detail, rankedHealthcheckTimeoutEnv)
}

func TestHostNodeRankedRuntimeHealthcheckReportsSkewAgainstTheCertifiedPin(t *testing.T) {
	// Satisfies the >=20.19.0 floor but is not the certified 22.23.2.
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("20.19.0", correctlyRootedAnswer))
	skewed := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessReady, skewed.Status, skewed.Detail)
	require.Equal(t, domain.ReasonRankedRuntimeVersionSkew, skewed.ReasonCode)
	require.Contains(t, skewed.Detail, "22.23.2")

	require.NoError(t, os.WriteFile(state.Node, []byte(fakeHostNode("22.23.2", correctlyRootedAnswer)), 0o755))
	matching := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessReady, matching.Status, matching.Detail)
	require.Equal(t, "ranked_runtime_healthy", matching.ReasonCode)
}

func TestHostNodeRankedRuntimeHealthcheckSeparatesAFailingNodeFromAnOldOne(t *testing.T) {
	strategist, runtimeRoot, state := hostRuntimeFixture(t, fakeHostNode("18.0.0", correctlyRootedAnswer))
	old := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessBlocked, old.Status)
	require.Equal(t, "ranked_runtime_host_node_unsupported", old.ReasonCode)

	require.NoError(t, os.WriteFile(state.Node, []byte("#!/bin/sh\nexit 3\n"), 0o755))
	got := runHostNodeRankedRuntimeHealthcheck(strategist, runtimeRoot, "openspec-propose", hostNodeContract(), state)
	require.Equal(t, domain.ReadinessBlocked, got.Status)
	require.Equal(t, "ranked_runtime_healthcheck_failed", got.ReasonCode,
		"a Node that runs but fails is an execution failure, not an unsupported version")
	require.Contains(t, got.Detail, "could not be executed")
}
