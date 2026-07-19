// Package stale implements the stale-artifact checker for compiled Strategist artifacts.
package stale

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Checker implements domain.StaleChecker.
type Checker struct{}

// IsStale reports whether the compiled artifact at artifactPath is absent, stale,
// or has a missing manifest. It mirrors the logic of check-stale.sh exactly:
//
//  1. Artifact file must exist.
//  2. .manifest.gz must exist in the same directory.
//  3. Every source file listed in artifact.sources must exist and have an mtime
//     no newer than the recorded value.
func (c Checker) IsStale(artifactPath string) (bool, error) {
	missing, err := pathMissing(artifactPath, "artifact")
	if missing || err != nil {
		return missing, err
	}

	manifestPath := filepath.Join(filepath.Dir(artifactPath), ".manifest.gz")
	missing, err = pathMissing(manifestPath, "manifest")
	if missing || err != nil {
		return missing, err
	}

	sources, err := readSources(artifactPath)
	if err != nil {
		return false, fmt.Errorf("stale: read sources: %w", err)
	}
	return sourcesAreStale(sources)
}

func pathMissing(path, label string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("stale: stat %s: %w", label, err)
	}
	return false, nil
}

func sourcesAreStale(sources map[string]int64) (bool, error) {
	for path, recorded := range sources {
		stale, err := sourceIsStale(path, recorded)
		if stale || err != nil {
			return stale, err
		}
	}
	return false, nil
}

func sourceIsStale(path string, recorded int64) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("stale: stat source %s: %w", path, err)
	}
	if info.ModTime().Unix() > recorded {
		return true, nil
	}
	return false, nil
}

// readSources decompresses a gzipped JSON artifact and extracts the sources map.
func readSources(artifactPath string) (map[string]int64, error) {
	f, err := os.Open(artifactPath) //nolint:gosec // G304: path comes from CLI caller — expected behavior
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close on read-only file

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close() //nolint:errcheck // best-effort close

	var artifact struct {
		Sources map[string]int64 `json:"sources"`
	}
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	if artifact.Sources == nil {
		return map[string]int64{}, nil
	}

	return artifact.Sources, nil
}

// Ensure Checker satisfies the domain interface at compile time.
var _ domain.StaleChecker = Checker{}
