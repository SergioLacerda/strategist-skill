// Package stale implements the stale-artifact checker for compiled Strategist artifacts.
package stale

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Checker implements domain.StaleChecker.
type Checker struct{}

// Check reports whether the compiled artifact is stale and why.
func (c Checker) Check(artifactPath string) (Result, error) {
	result := Result{ArtifactPath: artifactPath, Reason: ReasonFresh}
	if missing, err := pathMissing(artifactPath, "artifact"); missing || err != nil {
		result.Stale = missing
		result.Reason = ReasonMissingArtifact
		return result, err
	}

	if staleResult, err := checkManifest(artifactPath, result); staleResult.Stale || err != nil {
		return staleResult, err
	}

	sources, err := readArtifactSources(artifactPath)
	if err != nil {
		return result, fmt.Errorf("stale: read sources: %w", err)
	}
	return checkArtifactSources(result, sources)
}

// IsStale preserves the legacy domain.StaleChecker API.
func (c Checker) IsStale(artifactPath string) (bool, error) {
	result, err := c.Check(artifactPath)
	if err != nil {
		return false, err
	}
	return result.Stale, nil
}

func pathMissing(path, label string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("stale: stat %s: %w", label, err)
	}
	return false, nil
}

func readGzJSON(path string, v any) error {
	f, err := os.Open(path) //nolint:gosec // G304: path comes from CLI caller or sibling manifest
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close on read-only file

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close() //nolint:errcheck // best-effort close

	if err := json.NewDecoder(gz).Decode(v); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}

// Ensure Checker satisfies the domain interface at compile time.
var _ domain.StaleChecker = Checker{}
