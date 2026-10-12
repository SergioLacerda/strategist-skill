package treasuretmp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
	"gopkg.in/yaml.v3"
)

const defaultRunbookSourcePath = "docs/runbooks"

type activeRunbookConfig struct {
	Runbooks struct {
		CanonicalPath string `yaml:"canonical_path"`
	} `yaml:"runbooks"`
}

type indexedRunbookSource struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type knowledgeIndexConfig struct {
	Sources []indexedRunbookSource `yaml:"sources"`
}

type runbookSourceLayers struct {
	activePath, manifestPath, indexPath             string
	activeDeclared, manifestDeclared, indexDeclared bool
}

func (l runbookSourceLayers) anyDeclared() bool {
	return l.activeDeclared || l.manifestDeclared || l.indexDeclared
}

func (l runbookSourceLayers) complete() bool {
	return l.activeDeclared && l.manifestDeclared && l.indexDeclared
}

func (l runbookSourceLayers) aligned() bool {
	return normalizeRunbookSourcePath(l.activePath) == normalizeRunbookSourcePath(l.manifestPath) &&
		normalizeRunbookSourcePath(l.activePath) == normalizeRunbookSourcePath(l.indexPath)
}

// validateRunbookSourceConsistency keeps the three declarations that govern
// runbook retrieval in one fail-closed contract. An incomplete declaration is
// drift, not permission to read another path.
func validateRunbookSourceConsistency(root string, governed map[string]treasure.GovernedChest) error {
	layers, err := readRunbookSourceLayers(root, governed)
	if err != nil {
		return err
	}
	if !layers.anyDeclared() {
		return nil
	}
	if !layers.complete() {
		return fmt.Errorf("active.yaml, treasure-chests.yaml, and knowledge.index.yaml must all declare runbooks")
	}
	if !layers.aligned() {
		return fmt.Errorf("active.yaml=%s, treasure-chests.yaml=%s, knowledge.index.yaml=%s",
			normalizeRunbookSourcePath(layers.activePath),
			normalizeRunbookSourcePath(layers.manifestPath),
			normalizeRunbookSourcePath(layers.indexPath))
	}
	return nil
}

func readRunbookSourceLayers(root string, governed map[string]treasure.GovernedChest) (runbookSourceLayers, error) {
	activePath, activeDeclared, err := readActiveRunbookPath(root)
	if err != nil {
		return runbookSourceLayers{}, err
	}
	indexPath, indexDeclared, err := readIndexedRunbookPath(root)
	if err != nil {
		return runbookSourceLayers{}, err
	}
	manifestPath, manifestDeclared := governedRunbookPath(governed)
	return runbookSourceLayers{
		activePath: activePath, manifestPath: manifestPath, indexPath: indexPath,
		activeDeclared: activeDeclared, manifestDeclared: manifestDeclared, indexDeclared: indexDeclared,
	}, nil
}

func readActiveRunbookPath(root string) (string, bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // root is the explicitly bound provider workspace
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg activeRunbookConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", false, fmt.Errorf("parse active.yaml: %w", err)
	}
	return cfg.Runbooks.CanonicalPath, strings.TrimSpace(cfg.Runbooks.CanonicalPath) != "", nil
}

func governedRunbookPath(governed map[string]treasure.GovernedChest) (string, bool) {
	chest, ok := governed["runbooks"]
	if !ok || strings.TrimSpace(chest.Path) == "" || chest.Status == "inactive" {
		return "", false
	}
	return chest.Path, true
}

func readIndexedRunbookPath(root string) (string, bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "knowledge.index.yaml")) //nolint:gosec // root is the explicitly bound provider workspace
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read knowledge.index.yaml: %w", err)
	}
	var cfg knowledgeIndexConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", false, fmt.Errorf("parse knowledge.index.yaml: %w", err)
	}
	if path, ok := indexedRunbookPath(cfg.Sources); ok {
		return path, true, nil
	}
	return "", false, nil
}

func indexedRunbookPath(sources []indexedRunbookSource) (string, bool) {
	for _, source := range sources {
		if source.ID == "runbooks" && strings.TrimSpace(source.Path) != "" {
			return source.Path, true
		}
	}
	return "", false
}

func normalizeRunbookSourcePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if path == "." || path == "" {
		return defaultRunbookSourcePath
	}
	return path
}
