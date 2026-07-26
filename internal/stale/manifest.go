package stale

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

type compiledManifest struct {
	Artifacts map[string]string `json:"artifacts"`
}

func checkManifest(artifactPath string, result Result) (Result, error) {
	manifestPath := filepath.Join(filepath.Dir(artifactPath), ".manifest.gz")
	if missing, err := pathMissing(manifestPath, "manifest"); missing || err != nil {
		result.Stale = missing
		result.Reason = ReasonMissingManifest
		return result, err
	}

	manifest, err := readManifest(manifestPath)
	if err != nil {
		return result, fmt.Errorf("stale: read manifest: %w", err)
	}
	if len(manifest.Artifacts) == 0 {
		result.Detail = "legacy manifest has no artifact checksums"
		return result, nil
	}

	name := filepath.Base(artifactPath)
	recorded, ok := manifest.Artifacts[name]
	if !ok {
		result.Stale = true
		result.Reason = ReasonManifestEntryMissing
		return result, nil
	}
	current := artifactSHA256(artifactPath)
	if current != recorded {
		result.Stale = true
		result.Reason = ReasonArtifactHashMismatch
		result.Detail = fmt.Sprintf("recorded=%s current=%s", recorded, current)
	}
	return result, nil
}

func readManifest(path string) (compiledManifest, error) {
	var manifest compiledManifest
	if err := readGzJSON(path, &manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func artifactSHA256(path string) string {
	hash, exists, err := runtimefs.ReadSHA256(path)
	if err != nil || !exists {
		return "unavailable"
	}
	return "sha256:" + hash
}
