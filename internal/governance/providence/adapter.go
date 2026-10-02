// Package providence adapts an explicitly selected Providence directory to
// governance.Source. Providence is an optional reference provider; the
// generic governance package does not depend on this package.
package providence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/governance"
)

const sourceName = "providence"

type source struct{}

type metadata struct {
	Fingerprints struct {
		Combined string `json:"combined"`
	} `json:"fingerprints"`
}

type governanceCore struct {
	Items []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	} `json:"items"`
}

// New returns the explicit Providence reference adapter. The directory is
// deliberately supplied later to Snapshot so callers cannot accidentally
// introduce a default or auto-detected governance path.
func New() governance.Source {
	return source{}
}

func (source) Name() string { return sourceName }

func (source) Snapshot(governanceDir string) (governance.Snapshot, error) {
	if governanceDir == "" {
		return governance.Snapshot{}, fmt.Errorf("providence source: governance directory is required")
	}

	meta, metadataPath, err := readMetadata(governanceDir)
	if err != nil {
		return governance.Snapshot{}, err
	}
	if meta.Fingerprints.Combined == "" {
		return governance.Snapshot{}, fmt.Errorf("providence source: %s has no fingerprints.combined", metadataPath)
	}

	core, corePath, err := readCore(governanceDir)
	if err != nil {
		return governance.Snapshot{}, err
	}
	mandates, err := requiredMandates(core, corePath)
	if err != nil {
		return governance.Snapshot{}, err
	}

	return governance.Snapshot{
		SourceID:       sourceName,
		Fingerprint:    meta.Fingerprints.Combined,
		ActiveMandates: mandates,
		Scope:          filepath.Clean(governanceDir),
		CorrelationID:  sourceName + ":" + meta.Fingerprints.Combined,
		Validated:      true,
	}, nil
}

func readMetadata(governanceDir string) (metadata, string, error) {
	path := filepath.Join(governanceDir, "metadata.json")
	var meta metadata
	if err := readJSON(path, &meta); err != nil {
		return metadata{}, path, fmt.Errorf("providence source: %w", err)
	}
	return meta, path, nil
}

func readCore(governanceDir string) (governanceCore, string, error) {
	path := filepath.Join(governanceDir, "source", "governance-core.json")
	var core governanceCore
	if err := readJSON(path, &core); err != nil {
		return governanceCore{}, path, fmt.Errorf("providence source: %w", err)
	}
	return core, path, nil
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path) //nolint:gosec // explicit caller-selected governance directory
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func requiredMandates(core governanceCore, corePath string) ([]string, error) {
	mandates := make([]string, 0, len(core.Items))
	for _, item := range core.Items {
		if item.Type != "MANDATE" || item.Status != "required" {
			continue
		}
		if item.ID == "" {
			return nil, fmt.Errorf("providence source: %s contains a required mandate without an id", corePath)
		}
		mandates = append(mandates, item.ID)
	}
	sort.Strings(mandates)
	return mandates, nil
}
