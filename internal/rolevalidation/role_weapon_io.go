package rolevalidation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func readRoleMap(root string) (domain.RoleSlotMap, error) {
	raw, err := os.ReadFile(filepath.Join(root, "roles", "default.yaml")) //nolint:gosec // fixed runtime path
	if err != nil {
		return nil, fmt.Errorf("read role slot map: %w", err)
	}
	var m domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal role slot map: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate role slot map: %w", err)
	}
	return m, nil
}

func readLock(root string) (domain.PluginLockFile, error) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins.lock")) //nolint:gosec // fixed runtime path
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("unmarshal plugins.lock: %w", err)
	}
	return lock, nil
}
