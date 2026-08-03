//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseHistoryAuthorityIsDocumented(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	changelog := readFile(t, filepath.Join(root, "CHANGELOG.md"))
	contributing := readFile(t, filepath.Join(root, "CONTRIBUTING.md"))
	runbook := readFile(t, filepath.Join(root, "docs", "runbooks", "local-ci-cd-release-gates.md"))

	for _, doc := range []struct {
		name    string
		content string
	}{
		{"CHANGELOG.md", changelog},
		{"CONTRIBUTING.md", contributing},
		{"docs/runbooks/local-ci-cd-release-gates.md", runbook},
	} {
		for _, needle := range []string{
			"GitHub Releases are authoritative",
			"patch releases after `1.0.0`",
		} {
			if !strings.Contains(doc.content, needle) {
				t.Fatalf("%s missing release-history authority term %q", doc.name, needle)
			}
		}
	}
	if strings.Contains(changelog, "## [1.0.8]") || strings.Contains(changelog, "## [1.0.9]") {
		t.Fatalf("CHANGELOG.md must not backfill unverified v1.0.x patch notes")
	}
}

func TestMonorepoAndToolchainPolicyADRExists(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	adr := readFile(t, filepath.Join(root, "docs", "adr", "0014-monorepo-and-toolchain-policy.md"))
	contributing := readFile(t, filepath.Join(root, "CONTRIBUTING.md"))

	for _, needle := range []string{
		"Keep the Go CLI, embedded defaults, docs, and landing site in one repository",
		"Go version authority is `go.mod`",
		"`go 1.26.4` is the module/language target",
		"`toolchain go1.26.5` is the exact patch toolchain",
		"Node 22 is the supported major version for `web/landing/`",
		"`release-verify`",
		"`ci-web`",
	} {
		if !strings.Contains(adr, needle) {
			t.Fatalf("ADR-0014 missing monorepo/toolchain policy term %q", needle)
		}
	}
	for _, needle := range []string{
		"Relax or bump these pins only through the toolchain policy in ADR-0014",
		"Node is intentionally scoped to `web/landing/`",
	} {
		if !strings.Contains(normalizeWhitespace(contributing), needle) {
			t.Fatalf("CONTRIBUTING.md missing toolchain policy term %q", needle)
		}
	}
}

func TestBehavioralScenarioResidualsAreCovered(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	forbidden := readFile(t, filepath.Join(root, "tests", "spec", "specs", "forbidden-behaviors.feature"))
	tokenEconomy := readFile(t, filepath.Join(root, "tests", "spec", "specs", "token-economy.feature"))
	execution := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "contracts", "narrative", "06-execution.md"))

	for _, needle := range []string{
		"Scenario: provider_unavailable_no_direct_fallback",
		"execution_provider_unavailable",
		"must not perform the materialization itself",
		"Scenario: model_variance_preserves_blocking_question",
		"unresolved question remains open",
	} {
		if !strings.Contains(forbidden, needle) {
			t.Fatalf("forbidden-behaviors.feature missing behavioral scenario term %q", needle)
		}
	}
	for _, needle := range []string{
		"Scenario: unresolved_question_not_promoted_to_decision",
		"does not record the missing answer as an accepted decision",
	} {
		if !strings.Contains(tokenEconomy, needle) {
			t.Fatalf("token-economy.feature missing open-question scenario term %q", needle)
		}
	}
	if !strings.Contains(execution, "Missing or uncallable provider is a blocked state, not a reason for direct execution.") {
		t.Fatalf("execution contract must forbid direct fallback when provider invocation fails")
	}
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestOTelContextPropagationContractIsDocumented(t *testing.T) {
	t.Parallel()

	telemetry := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "narrative", "10-telemetry.md"))
	for _, needle := range []string{
		"CLI commands that create spans must start from the Cobra command context via",
		"Install, compile, check, validate, and sync-governance spans must preserve",
		"Telemetry setup and shutdown may use `context.Background()`",
		"When OTLP is disabled, the no-op tracer provider must keep the same context propagation behavior",
	} {
		if !strings.Contains(telemetry, needle) {
			t.Fatalf("telemetry contract missing OTel context propagation term %q", needle)
		}
	}
}
