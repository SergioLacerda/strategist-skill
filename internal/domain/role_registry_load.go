package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadRoleRegistry overlays the roles/*.yaml definitions found in dir on the
// built-in registry: a file replaces the role with the same id, and a new file
// adds a role. A missing directory yields the built-ins; a malformed file is an
// error. default.yaml (the slot map) is not a role definition.
func LoadRoleRegistry(dir string) (RoleRegistry, error) {
	configs, err := readRoleConfigs(dir)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultRoleRegistry(), nil
	}
	if err != nil {
		return RoleRegistry{}, err
	}
	merged := map[string]Role{}
	for _, role := range DefaultRoleRegistry().roles {
		merged[role.ID] = role
	}
	for _, cfg := range configs {
		role := RoleFromConfig(cfg)
		role.ID = normalizeRoleID(role.ID)
		merged[role.ID] = role
	}
	roles := make([]Role, 0, len(merged))
	for _, role := range merged {
		roles = append(roles, role)
	}
	return NewRoleRegistry(roles)
}

// readRoleConfigs parses every role definition file in dir.
func readRoleConfigs(dir string) ([]RoleConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("role registry: read %s: %w", dir, err)
	}
	var configs []RoleConfig
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

func parseRoleFile(path string) (RoleConfig, error) {
	name := filepath.Base(path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixed directory below .strategist
	if err != nil {
		return RoleConfig{}, fmt.Errorf("role registry: read %s: %w", name, err)
	}
	var cfg RoleConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return RoleConfig{}, fmt.Errorf("role registry: parse %s: %w", name, err)
	}
	if normalizeRoleID(cfg.Role) == "" {
		return RoleConfig{}, fmt.Errorf("role registry: %s: role is required", name)
	}
	return cfg, nil
}
