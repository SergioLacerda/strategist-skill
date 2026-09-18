package compile

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// VerifyManifest recomputes the SHA256 of each artifact recorded in
// <compiledDir>/.manifest.gz and reports a drift description for every
// mismatch or missing artifact file. If .manifest.gz itself is absent, it
// returns a single "no manifest" drift note rather than an error — this is a
// detectable-but-not-catastrophic condition (e.g. a pre-manifest install).
func VerifyManifest(compiledDir string) ([]string, error) {
	manifestPath := filepath.Join(compiledDir, ".manifest.gz")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return []string{fmt.Sprintf("manifest_drift: %s not found — run strategist compile", manifestPath)}, nil
	}

	manifest, err := readManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("verify manifest: read %s: %w", manifestPath, err)
	}

	return manifestDrift(compiledDir, manifest)
}

func manifestDrift(compiledDir string, manifest compiledManifest) ([]string, error) {
	names := sortedManifestNames(manifest)
	var drift []string
	for _, name := range names {
		if err := validateManifestName(compiledDir, name); err != nil {
			return nil, fmt.Errorf("verify manifest: %w", err)
		}
		if message := artifactDrift(compiledDir, name, manifest.Artifacts[name]); message != "" {
			drift = append(drift, message)
		}
	}
	return drift, nil
}

func sortedManifestNames(manifest compiledManifest) []string {
	names := make([]string, 0, len(manifest.Artifacts))
	for name := range manifest.Artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func artifactDrift(compiledDir, name, recorded string) string {
	artifactPath := filepath.Join(compiledDir, name)
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return fmt.Sprintf("manifest_drift: artifact %s missing (recorded in manifest)", name)
	}
	current := sha256Artifact(artifactPath)
	if current != recorded {
		return fmt.Sprintf("manifest_drift: artifact %s hash mismatch — recorded=%s current=%s", name, recorded, current)
	}
	return ""
}

func validateManifestName(compiledDir, name string) error {
	if name == "" || filepath.IsAbs(name) || filepath.Clean(name) != name || name == "." || name == ".." {
		return fmt.Errorf("manifest entry %q is not a clean relative path", name)
	}
	rel, err := filepath.Rel(compiledDir, filepath.Join(compiledDir, name))
	if err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return fmt.Errorf("manifest entry %q escapes compiled runtime", name)
	}
	return nil
}

// readManifest decompresses and decodes a .manifest.gz file.
func readManifest(path string) (compiledManifest, error) {
	var manifest compiledManifest

	f, err := os.Open(path) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return manifest, fmt.Errorf("open: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close on read-only file

	gz, err := gzip.NewReader(f)
	if err != nil {
		return manifest, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close() //nolint:errcheck // best-effort close

	if err := json.NewDecoder(gz).Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("json decode: %w", err)
	}

	return manifest, nil
}
