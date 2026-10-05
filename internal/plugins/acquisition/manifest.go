package acquisition

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	manifestSchemaVersion   = "strategist-acquisition/v1"
	acquisitionManifestName = "acquisition.json"
	readyMarkerName         = "READY"
)

// ManifestFile is one deterministic package-tree entry recorded for review.
type ManifestFile struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Kind   string `json:"kind"`
	Bytes  int64  `json:"bytes"`
	Digest string `json:"digest"`
}

// Manifest is provenance evidence only; trust.Verify remains the authority
// for whether a package may be promoted or installed.
type Manifest struct {
	SchemaVersion string         `json:"schema_version"`
	Source        string         `json:"source"`
	Repository    string         `json:"repository"`
	Commit        string         `json:"commit"`
	PackageID     string         `json:"package_id"`
	TreeDigest    string         `json:"tree_digest"`
	Files         []ManifestFile `json:"files"`
	AcquiredAt    string         `json:"acquired_at"`
	Validation    string         `json:"validation"`
}

func writeManifest(root string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return acquisitionError("manifest_write_failed", err.Error())
	}
	data = append(data, '\n')
	tmpName, err := createManifestTemp(root, data)
	if err != nil {
		return err
	}
	defer removeManifestTemp(tmpName) //nolint:gosec // exact temporary file created above
	if err := os.Rename(tmpName, filepath.Join(root, acquisitionManifestName)); err != nil {
		return acquisitionError("manifest_write_failed", err.Error())
	}
	return nil
}

func createManifestTemp(root string, data []byte) (tmpName string, err error) {
	tmp, err := os.CreateTemp(root, ".acquisition.json-*")
	if err != nil {
		return "", acquisitionError("manifest_write_failed", err.Error())
	}
	tmpName = tmp.Name()
	defer func() {
		if closeErr := tmp.Close(); closeErr != nil {
			err = acquisitionError("manifest_write_failed", closeErr.Error())
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return "", acquisitionError("manifest_write_failed", err.Error())
	}
	if _, err := tmp.Write(data); err != nil {
		return "", acquisitionError("manifest_write_failed", err.Error())
	}
	return tmpName, nil
}

func removeManifestTemp(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return
	}
}

func writeReady(root string) error {
	path := filepath.Join(root, readyMarkerName)
	if err := os.WriteFile(path, []byte("ready\n"), 0o600); err != nil {
		return acquisitionError("ready_write_failed", err.Error())
	}
	return nil
}

func newManifest(req Request, digest string, files []ManifestFile, now time.Time) Manifest {
	return Manifest{
		SchemaVersion: manifestSchemaVersion,
		Source:        req.Source,
		Repository:    req.Repository,
		Commit:        req.Commit,
		PackageID:     req.PackageID,
		TreeDigest:    digest,
		Files:         files,
		AcquiredAt:    now.UTC().Format(time.RFC3339),
		Validation:    "package_shape_validated;trust_pending",
	}
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != manifestSchemaVersion || manifest.Source != SourceGit {
		return fmt.Errorf("invalid acquisition manifest")
	}
	return nil
}
