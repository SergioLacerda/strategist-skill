package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	"gopkg.in/yaml.v3"
)

func loadMissionInvocationState(root string) (domain.ActiveConfig, domain.PluginLockFile, domain.CompiledRegistry, error) {
	activeRaw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read active.yaml: %w", err)
	}
	var active domain.ActiveConfig
	if err := yaml.Unmarshal(activeRaw, &active); err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse active.yaml: %w", err)
	}
	lockRaw, err := os.ReadFile(filepath.Join(root, "plugins.lock")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(lockRaw, &lock); err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse plugins.lock: %w", err)
	}
	catalogRaw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read compiled catalog: %w", err)
	}
	registry, err := catalog.ParseCompiledRegistryCatalog(catalogRaw)
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse compiled catalog: %w", err)
	}
	if err := requireRegistryMatchesBinary(registry); err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, err
	}
	return active, lock, registry, nil
}

// requireRegistryMatchesBinary anchors the Weapon/Role/binding registry to the
// catalog compiled into this binary: the workspace copy is user-writable, so a
// divergent registry is rejected instead of trusted.
func requireRegistryMatchesBinary(workspace domain.CompiledRegistry) error {
	raw, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	if err != nil {
		return fmt.Errorf("read embedded catalog: %w", err)
	}
	embedded, err := catalog.ParseCompiledRegistryCatalog(raw)
	if err != nil {
		return fmt.Errorf("parse embedded catalog: %w", err)
	}
	if !reflect.DeepEqual(embedded, workspace) {
		return fmt.Errorf("compiled_registry_drift: the workspace plugins/catalog.yaml registry differs from the one compiled into this binary; run `strategist upgrade` to restore it")
	}
	return nil
}
