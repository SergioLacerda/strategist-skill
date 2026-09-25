package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// PrepareEmbeddedOptions configures a strategist plugin prepare-embedded run
// — the maintainer/CI entry point for ingesting external-skills-source/ into
// this repository's own embedded plugin defaults (tasks.md Task 3).
type PrepareEmbeddedOptions struct {
	Source       string
	DefaultsRoot string
	LockPath     string
}

func (o PrepareEmbeddedOptions) catalogPath() string {
	return filepath.Join(o.DefaultsRoot, "plugins", "catalog.yaml")
}

// PrepareEmbeddedReport summarizes one PrepareEmbedded/CheckEmbeddedDrift run.
type PrepareEmbeddedReport struct {
	Ingested []IngestedSkill
	Rejected []IngestionRejection
}

// PrepareEmbedded ingests opts.Source and writes the merged catalog, mirrors,
// and lock to disk. Any rejection is reported, never silently dropped, but a
// rejection does not itself fail the run — only a hard I/O or parse error
// does. Callers that want CI to fail on any rejection should inspect
// report.Rejected themselves (see plugins_prepare_embedded.go).
func PrepareEmbedded(opts PrepareEmbeddedOptions) (PrepareEmbeddedReport, error) {
	result, err := ingestForOptions(opts)
	if err != nil {
		return PrepareEmbeddedReport{}, err
	}
	if err := WriteCatalogAndMirrors(result, opts.DefaultsRoot, opts.catalogPath(), opts.LockPath); err != nil {
		return PrepareEmbeddedReport{}, err
	}
	return PrepareEmbeddedReport{Ingested: result.Ingested, Rejected: result.Rejected}, nil
}

// CheckEmbeddedDrift computes what PrepareEmbedded would write and compares
// it against what is currently on disk, without writing anything — tasks.md
// Task 3.2/4.2: fail non-zero on lock/artifact/catalog drift, for CI.
func CheckEmbeddedDrift(opts PrepareEmbeddedOptions) (report PrepareEmbeddedReport, drift bool, err error) {
	result, err := ingestForOptions(opts)
	if err != nil {
		return PrepareEmbeddedReport{}, false, err
	}
	report = PrepareEmbeddedReport{Ingested: result.Ingested, Rejected: result.Rejected}

	tmpRoot, err := os.MkdirTemp("", "strategist-embed-check-*")
	if err != nil {
		return report, false, fmt.Errorf("check embedded drift: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpRoot); removeErr != nil && err == nil {
			err = fmt.Errorf("check embedded drift: remove temp dir: %w", removeErr)
		}
	}()

	drift, err = compareEmbeddedArtifacts(opts, result, tmpRoot)
	if err != nil {
		return report, false, err
	}
	drift = drift || len(result.Rejected) > 0
	return report, drift, nil
}

func compareEmbeddedArtifacts(opts PrepareEmbeddedOptions, result IngestionResult, tmpRoot string) (bool, error) {
	tmpCatalogPath := filepath.Join(tmpRoot, "catalog.yaml")
	tmpLockPath := filepath.Join(tmpRoot, "lock.yaml")
	if err := WriteCatalogAndMirrors(result, tmpRoot, tmpCatalogPath, tmpLockPath); err != nil {
		return false, fmt.Errorf("check embedded drift: %w", err)
	}
	catalogDrift, err := catalogHasDrifted(opts.catalogPath(), tmpCatalogPath)
	if err != nil {
		return false, err
	}
	mirrorsDrift, err := mirrorsHaveDrifted(result.Ingested, opts.DefaultsRoot, tmpRoot)
	if err != nil {
		return false, err
	}
	lockDrift, err := lockHasDrifted(opts.LockPath, tmpLockPath, len(result.Ingested) > 0)
	if err != nil {
		return false, err
	}
	return catalogDrift || mirrorsDrift || lockDrift, nil
}

func ingestForOptions(opts PrepareEmbeddedOptions) (IngestionResult, error) {
	existingCatalogBytes, err := os.ReadFile(opts.catalogPath()) //nolint:gosec // G304: fixed, caller-configured repo-relative path
	if err != nil {
		return IngestionResult{}, fmt.Errorf("read %s: %w", opts.catalogPath(), err)
	}
	existingCatalog, err := parseCatalogBytes(existingCatalogBytes)
	if err != nil {
		return IngestionResult{}, fmt.Errorf("parse %s: %w", opts.catalogPath(), err)
	}
	result, err := IngestExternalSkills(opts.Source, existingCatalog, domain.TrustPolicy{})
	if err != nil {
		return IngestionResult{}, fmt.Errorf("ingest external skills: %w", err)
	}
	if err := certifyRankedCandidates(&result.Catalog, opts.DefaultsRoot); err != nil {
		return IngestionResult{}, fmt.Errorf("certify ranked candidates: %w", err)
	}
	return result, nil
}
