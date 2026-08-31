package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
)

// runStrictChecks composes the additional checks --strict adds on top of the
// base check: required compiled artifacts must exist, and their recorded
// manifest hashes must match the artifacts on disk.
func runStrictChecks(root string) []string {
	var errs []string
	compiledDir := filepath.Join(root, ".compiled")

	requiredArtifacts := []string{".index.gz", ".domain.gz", ".config.gz", ".manifest.gz"}
	for _, name := range requiredArtifacts {
		artifactPath := filepath.Join(compiledDir, name)
		if _, statErr := os.Stat(artifactPath); os.IsNotExist(statErr) {
			errs = append(errs, fmt.Sprintf("strict: missing compiled artifact %s — run strategist compile", name))
		}
	}

	drift, err := compile.VerifyManifest(compiledDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("strict: verify manifest: %v", err))
	}
	for _, d := range drift {
		errs = append(errs, "strict: "+d)
	}

	return errs
}

func validateRuntimeDefaultParity(root string) []string {
	extractor := embedpkg.Extractor{}
	var errs []string
	manifest, manifestLoaded, manifestErr := readInstallManifest(root)
	if manifestErr != nil {
		errs = append(errs, fmt.Sprintf("runtime_stale: install manifest unreadable: %v", manifestErr))
	}

	for _, rel := range domain.NormativeRuntimeDefaultPaths() {
		err, ok := validateRuntimeDefaultFile(root, rel, extractor, manifest, manifestLoaded, manifestErr)
		if ok {
			errs = append(errs, err)
		}
	}

	return errs
}

func validateRuntimeDefaultFile(
	root, rel string,
	extractor embedpkg.Extractor,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	manifestErr error,
) (string, bool) {
	runtimePath := filepath.Join(root, filepath.FromSlash(rel))
	runtimeRaw, readErr := os.ReadFile(runtimePath) //nolint:gosec // G304: runtime file path is derived from the discovered .strategist root
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", false
		}
		return fmt.Sprintf("runtime_stale: read %s: %v", runtimePath, readErr), true
	}

	embeddedRaw, embedErr := extractor.ReadFile(rel)
	if embedErr != nil {
		return fmt.Sprintf("runtime_stale: embedded default %q unreadable: %v", rel, embedErr), true
	}
	if string(runtimeRaw) == string(embeddedRaw) {
		// No drift: exact byte identity with the embedded default. This check
		// only ever covers the byte drift class (and, on mismatch, provenance
		// via classifyRuntimeStale's manifest SHA256 lookup below) — see
		// docs/drift-detection-matrix.md.
		return "", false
	}
	return domain.FormatRuntimeStaleDiagnostic(
		rel,
		classifyRuntimeStale(runtimeRaw, rel, manifest, manifestLoaded, manifestErr),
	), true
}

func classifyRuntimeStale(
	runtimeRaw []byte,
	rel string,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	manifestErr error,
) domain.RuntimeDefaultDecision {
	if manifestErr != nil || !manifestLoaded {
		return domain.RuntimeDecisionUnknownManifest
	}
	manifestFile, ok := manifest.FileByPath(rel)
	if !ok {
		return domain.RuntimeDecisionUnknownManifest
	}
	if domain.SHA256Hex(runtimeRaw) == manifestFile.SHA256 {
		return domain.RuntimeDecisionAutoUpgrade
	}
	return domain.RuntimeDecisionConflict
}

func readInstallManifest(root string) (domain.InstallManifest, bool, error) {
	path := filepath.Join(root, domain.InstallManifestRelPath)
	data, err := os.ReadFile(path) //nolint:gosec // G304: runtime file path is derived from the discovered .strategist root
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.InstallManifest{}, false, nil
		}
		return domain.InstallManifest{}, false, fmt.Errorf("read install manifest: %w", err)
	}
	var manifest domain.InstallManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return domain.InstallManifest{}, false, fmt.Errorf("parse install manifest: %w", err)
	}
	return manifest, true, nil
}
