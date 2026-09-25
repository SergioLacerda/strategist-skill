package check

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// directivesRelPath mirrors identityRelPaths' convention (check_identity.go)
// for the sibling directives_missing condition in
// contracts/machine/preflight.yaml.
var directivesRelPath = filepath.Join("templates", "domain", "directives", "core.yaml")

// preflightAdvisories detects the non-blocking preflight.yaml conditions
// that are not check_identity.go's stricter identity_files_missing sibling:
// index_yaml_not_found, compiled_artifact_corrupt, and directives_missing.
// All three are documented in preflight.yaml as "Non-blocking... Continue" —
// this function only makes them observable in PreflightResult.Warnings; it
// never affects `strategist check`'s exit code or PreflightResult.Status
// (see buildPreflightResult, which appends these after status is already
// decided from the blocking warnings list).
func preflightAdvisories(root string) []string {
	advisories := domainIndexAdvisories(root)
	if _, err := os.Stat(filepath.Join(root, directivesRelPath)); os.IsNotExist(err) {
		advisories = append(advisories, fmt.Sprintf(
			"[Strategist] phase=preflight status=warn reason=directives_missing path=%s (continuing without behavioral directives)",
			filepath.ToSlash(directivesRelPath)))
	}
	return append(advisories, layoutSkewAdvisories(root)...)
}

// domainIndexAdvisories implements preflight.yaml's index_yaml_not_found and
// compiled_artifact_corrupt conditions. The compiled artifact, when present,
// takes priority (mirrors the fast_path/standard_path preference documented
// in preflight.yaml and .strategist/contracts/narrative/01-bootstrap.md) —
// index.yaml absence is only reported when there is no compiled artifact to
// fall back on.
func domainIndexAdvisories(root string) []string {
	domainGz := filepath.Join(root, ".compiled", ".domain.gz")
	if _, err := os.Stat(domainGz); err == nil {
		if corruptErr := verifyGzJSONParses(domainGz); corruptErr != nil {
			return []string{fmt.Sprintf(
				"[Strategist] phase=preflight status=warn reason=compiled_artifact_corrupt artifact=%s: %v (preflight=standard_path)",
				domainGz, corruptErr)}
		}
		return nil
	}
	indexPath := filepath.Join(root, "index.yaml")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return []string{fmt.Sprintf(
			"[Strategist] phase=preflight status=warn reason=index_yaml_not_found path=%s (continuing without internal domain)",
			indexPath)}
	}
	return nil
}

// verifyGzJSONParses confirms a gzip+JSON compiled artifact decodes cleanly,
// without interpreting its contents — the same shape written by
// internal/compile/domain.go's writeGzJSON and read by
// check_content_lang.go's readCompiledContentArtifact for the sibling
// .config.gz artifact.
func verifyGzJSONParses(path string) error {
	f, err := os.Open(path) //nolint:gosec // G304: fixed path under the runtime .compiled dir
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read-only file; close error is not actionable

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader %s: %w", path, err)
	}
	defer func() { _ = gz.Close() }() //nolint:errcheck // read-only reader; close error is not actionable

	var v any
	if err := json.NewDecoder(gz).Decode(&v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
