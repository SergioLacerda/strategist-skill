package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// nativeConnectorSourcePath and roleConformanceTestPath are repo-root-
// relative paths to the Strategist tool's own implementation evidence for
// ADR-0043 DEC-006's generic Ranked certification: the native-role
// dispatch mechanism, and the shared role/provider handoff-conformance
// test suite. Both are Strategist-internal source files compiled into the
// strategist binary — they never materialize into a target workspace's
// .strategist/ tree, unlike a role's own contract files
// (roles/<role>.yaml, internal_skills/<role>/SKILL.md — see
// hostAPIContractDigest). Resolved against repoRoot(), not the caller's
// working directory — see repoRoot's own doc comment for why.
const (
	nativeConnectorSourcePath = "internal/plugins/connectors/runtime_connector.go"
	roleConformanceTestPath   = "internal/handoff/role_provider_conformance_test.go"
)

// repoRoot resolves this Strategist repository's own root directory via
// runtime.Caller, independent of the caller's working directory. Unlike
// PrepareEmbeddedOptions.DefaultsRoot's CWD-relative default (which assumes
// invocation from the repo root, true for `go run ./cmd/strategist ...`),
// connectorDigest/testSuiteDigest must also resolve correctly under
// `go test ./internal/install/...`, whose working directory is this
// package's own directory, not the repo root.
func repoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile is <repoRoot>/internal/install/embedded_skill_conformance.go.
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// hostAPIContractDigest computes the ADR-0043 DEC-006 "host API contract"
// evidence for role: a sha256 digest over role's own contract files —
// roles/<role>.yaml and internal_skills/<role>/SKILL.md — read from
// defaultsRoot (the embedded-defaults authoring tree prepare-embedded
// already operates on). Generic over role: no role identity is
// special-cased in this function, unlike rankedCertificationPairs' own
// explicit allow-list.
func hostAPIContractDigest(defaultsRoot, role string) (string, error) {
	return digestFiles(
		filepath.Join(defaultsRoot, "roles", role+".yaml"),
		filepath.Join(defaultsRoot, "internal_skills", role, "SKILL.md"),
	)
}

// connectorDigest computes the ADR-0043 DEC-006 "connector" evidence: a
// digest over the native-role dispatch mechanism's own source
// (connectors.NativeRuntimeConnector). This is intentionally the same
// value for every role certified this way — every native role shares the
// same dispatch connector; the evidence is "this is the connector
// mechanism in effect," not a per-role variant.
func connectorDigest() (string, error) {
	return digestFiles(filepath.Join(repoRoot(), nativeConnectorSourcePath))
}

// testSuiteDigest computes the ADR-0043 DEC-006 "test suite" evidence: a
// digest over the shared role/provider handoff-conformance test file. It
// pins WHICH suite backs the certification claim, for staleness detection
// via conformance.CertificationRecord.Stale — it does not itself execute
// the suite. Running `go test` from inside a maintainer CLI step was
// judged out of proportion to what this pilot needs; see
// .analysis/pending/cli_refactor/20260916-conformance-wiring-and-adr0029-t2-decisions/design.md.
//
// Unlike connectorDigest (KF-006/docs/adr/0045: genuinely role-agnostic
// content, since every native role shares the same dispatch connector),
// this pin is NOT honestly role-agnostic: role_provider_conformance_test.go's
// actual test content (as of 2026-09-16) exercises
// handoff.RangerToArchivistPolicy() specifically — it verifies Ranger's own
// outgoing-handoff contract, not a generic "any role" suite. Every Ranked
// pairing today, regardless of role, is stamped with this same digest — it
// functions as a pinned code-freshness marker (this file changing
// invalidates every existing certification, whichever role it names) not
// as a role-specific contract-test guarantee. A pairing whose role is not
// Ranger (e.g. Archivist↔openspec-propose, docs/adr/0045) inherits this
// same accepted limitation. A real per-role contract test suite, plus
// making this function role-parameterized like hostAPIContractDigest, is a
// named, separate follow-up:
// .analysis/pending/20260916-archivist-conformance-test-suite.md.
func testSuiteDigest() (string, error) {
	return digestFiles(filepath.Join(repoRoot(), roleConformanceTestPath))
}

func digestFiles(paths ...string) (string, error) {
	h := sha256.New()
	for _, p := range paths {
		data, err := os.ReadFile(p) //nolint:gosec // G304: fixed, maintainer-controlled repo-relative paths
		if err != nil {
			return "", fmt.Errorf("digest %s: %w", p, err)
		}
		h.Write(data)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
