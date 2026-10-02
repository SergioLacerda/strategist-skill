package check

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
)

// directivesRelPath mirrors identityRelPaths' convention (check_identity.go)
// for the sibling directives_missing condition in
// contracts/machine/preflight.yaml.
var directivesRelPath = filepath.Join("templates", "domain", "directives", "core.yaml")

// preflightAdvisories detects non-blocking preflight conditions that are not
// check_identity.go's stricter identity_files_missing sibling, plus client
// bootstrap advisories such as CODEX seed drift. These conditions are
// documented as "Non-blocking... Continue" — this function only makes them
// observable in PreflightResult.Warnings; it never affects `strategist check`'s
// exit code or PreflightResult.Status
// (see buildPreflightResult, which appends these after status is already
// decided from the blocking warnings list).
func preflightAdvisories(root string) []string {
	advisories := domainIndexAdvisories(root)
	if _, err := os.Stat(filepath.Join(root, directivesRelPath)); os.IsNotExist(err) {
		advisories = append(advisories, fmt.Sprintf(
			"[Strategist] phase=preflight status=warn reason=directives_missing path=%s (continuing without behavioral directives)",
			filepath.ToSlash(directivesRelPath)))
	}
	advisories = append(advisories, registryDriftAdvisories(root)...)
	advisories = append(advisories, layoutSkewAdvisories(root)...)
	advisories = append(advisories, compatViewResidualAdvisories(root)...)
	return append(advisories, codexBootstrapAdvisories(root)...)
}

// compatViewResidualAdvisories reports cataloged Weapons whose old id-only
// compatibility view remains in a generation-2 workspace. The file is never
// read for authority and is never deleted automatically; this is visibility
// only, so residual state cannot change readiness or the check exit status.
func compatViewResidualAdvisories(root string) []string {
	if domain.RuntimeLayoutGeneration < 2 {
		return nil
	}
	facts, err := domain.ListCatalogWeaponFacts(root)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var advisories []string
	for _, fact := range facts {
		if advisory := compatViewResidualAdvisory(root, fact, seen); advisory != "" {
			advisories = append(advisories, advisory)
		}
	}
	return advisories
}

func compatViewResidualAdvisory(root string, fact domain.WeaponFacts, seen map[string]bool) string {
	if fact.CompatibilitySource == "native_role" || seen[fact.ID] {
		return ""
	}
	seen[fact.ID] = true
	path := filepath.Join(root, "skills", fact.ID, "skill.yaml")
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return fmt.Sprintf(
		"[Strategist] phase=preflight status=warn reason=compat_view_residual provider=%s path=%s (the generation-2 compatibility view is non-authoritative and edits no longer affect resolution)",
		fact.ID, path)
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

// registryDriftAdvisories reports a workspace plugins/catalog.yaml whose
// compiled registry differs from the one built into this binary. Mission
// invocation refuses such a workspace (compiled_registry_drift), so preflight
// surfaces it early. A missing catalog is reported by the other checks.
func registryDriftAdvisories(root string) []string {
	return registryDriftAdvisoriesAgainst(root, embeddedCatalogReader)
}

// embeddedCatalogReader yields the catalog compiled into this binary; tests
// substitute it to keep fixture workspaces independent of the shipped registry.
var embeddedCatalogReader = func() ([]byte, error) {
	return embedpkg.Extractor{}.ReadFile("plugins/catalog.yaml")
}

func registryDriftAdvisoriesAgainst(root string, readEmbedded func() ([]byte, error)) []string {
	workspaceRaw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return nil
	}
	embeddedRaw, err := readEmbedded()
	if err != nil {
		return nil
	}
	drifted, err := domain.CompiledRegistryDrift(workspaceRaw, embeddedRaw)
	if err != nil {
		return []string{fmt.Sprintf("[Strategist] phase=preflight status=warn reason=compiled_registry_unreadable: %v (run `strategist upgrade`)", err)}
	}
	if !drifted {
		return nil
	}
	return []string{"[Strategist] phase=preflight status=warn reason=compiled_registry_drift path=plugins/catalog.yaml (the workspace registry differs from this binary; mission invoke will refuse it — run `strategist upgrade`)"}
}
