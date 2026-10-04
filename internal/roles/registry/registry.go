// Package roles adapts workspace role manifests into the domain registry.
package roles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadRoleRegistry overlays roles/*.yaml on built-in role facts. The adapter
// owns filesystem and YAML concerns; domain.RoleRegistryFromConfigs owns merge
// and validation policy.
func LoadRoleRegistry(dir string) (domain.RoleRegistry, error) {
	configs, err := readRoleConfigs(dir)
	if errors.Is(err, os.ErrNotExist) {
		return domain.DefaultRoleRegistry(), nil
	}
	if err != nil {
		return domain.RoleRegistry{}, err
	}
	registry, err := domain.RoleRegistryFromConfigs(configs)
	if err != nil {
		return domain.RoleRegistry{}, fmt.Errorf("validate role registry: %w", err)
	}
	return registry, nil
}

func readRoleConfigs(dir string) ([]domain.RoleConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("role registry: read %s: %w", dir, err)
	}
	var configs []domain.RoleConfig
	for _, entry := range entries {
		if !isRoleFile(entry) {
			continue
		}
		cfg, err := parseRoleFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func isRoleFile(entry os.DirEntry) bool {
	name := entry.Name()
	return !entry.IsDir() && filepath.Ext(name) == ".yaml" && name != "default.yaml"
}

func parseRoleFile(path string) (domain.RoleConfig, error) {
	name := filepath.Base(path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixed directory below .strategist
	if err != nil {
		return domain.RoleConfig{}, fmt.Errorf("role registry: read %s: %w", name, err)
	}
	var cfg domain.RoleConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("role registry: parse %s: %w", name, err)
	}
	if cfg.Role == "" {
		return domain.RoleConfig{}, fmt.Errorf("role registry: %s: role is required", name)
	}
	return cfg, nil
}
