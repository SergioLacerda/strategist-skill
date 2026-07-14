package compile

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	var drift []string
	for name, recorded := range manifest.Artifacts {
		artifactPath := filepath.Join(compiledDir, name)
		if _, statErr := os.Stat(artifactPath); os.IsNotExist(statErr) {
			drift = append(drift, fmt.Sprintf("manifest_drift: artifact %s missing (recorded in manifest)", name))
			continue
		}
		current := sha256Artifact(artifactPath)
		if current != recorded {
			drift = append(drift, fmt.Sprintf("manifest_drift: artifact %s hash mismatch — recorded=%s current=%s", name, recorded, current))
		}
	}

	return drift, nil
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
