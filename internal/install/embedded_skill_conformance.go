package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// nativeConnectorSourcePath and roleConformanceTestPaths are repo-root-
// relative paths to the Strategist tool's own implementation evidence for
// ADR-0043 DEC-006's generic Ranked certification: the native-role
// dispatch mechanism, and the per-role handoff-conformance test suites. Both
// are Strategist-internal source files compiled into the
// strategist binary — they never materialize into a target workspace's
// .strategist/ tree, unlike a role's own contract files
// (roles/<role>.yaml, internal_skills/<role>/SKILL.md — see
// hostAPIContractDigest). Resolved against repoRoot(), not the caller's
// working directory — see repoRoot's own doc comment for why.
const (
	nativeConnectorSourcePath = "internal/plugins/connectors/runtime_connector.go"
	policySourcePath          = "internal/plugins/policy/grants.go"
	policyEnforcementPath     = "internal/plugins/policy/write_enforcement.go"
)

// roleConformanceTestPaths maps each Ranked-certifiable role to its own
// conformance test file. A separate file per role keeps the certification
// digest role-specific rather than pinning duplicated evidence.
var roleConformanceTestPaths = map[string]string{
	"ranger":    "internal/handoff/role_provider_conformance_test.go",
	"archivist": "internal/handoff/role_provider_conformance_archivist_test.go",
	"sniper":    "internal/handoff/role_provider_conformance_sniper_test.go",
}

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

// policyDigest binds Ranked certification to the permission and write-policy
// implementation that governs plugin activation and enforcement.
func policyDigest() (string, error) {
	return digestFiles(filepath.Join(repoRoot(), policySourcePath), filepath.Join(repoRoot(), policyEnforcementPath))
}

// testSuiteDigest computes the ADR-0043 DEC-006 "test suite" evidence for
// role: a digest over that role's own handoff-conformance test file. It pins
// WHICH suite backs the certification claim, for staleness detection
// via conformance.CertificationRecord.Stale — it does not itself execute
// the suite. Running `go test` from inside a maintainer CLI step was
// judged out of proportion to what this pilot needs; see
// .analysis/pending/cli_refactor/20260916-conformance-wiring-and-adr0029-t2-decisions/design.md.
// Generic over role: adding a Ranked pairing for a new role means adding
// its own entry to roleConformanceTestPaths, not sharing another role's
// evidence file.
func testSuiteDigest(role string) (string, error) {
	path, ok := roleConformanceTestPaths[role]
	if !ok {
		return "", fmt.Errorf("test suite digest: no conformance test file pinned for role %q", role)
	}
	return digestFiles(filepath.Join(repoRoot(), path))
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
