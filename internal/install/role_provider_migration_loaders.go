package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func loadRoleSlotMap(extractor domain.FileExtractor) (domain.RoleSlotMap, error) {
	data, err := extractor.ReadFile(roleSlotMapPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", roleSlotMapPath, err)
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(data, &roleMap); err != nil {
		return nil, fmt.Errorf("%s: %w", roleSlotMapPath, err)
	}
	if err := roleMap.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", roleSlotMapPath, err)
	}
	return roleMap, nil
}

func loadRoleConfig(extractor domain.FileExtractor, roleName string) (domain.RoleConfig, error) {
	path := "roles/" + roleName + ".yaml"
	data, err := extractor.ReadFile(path)
	if err != nil {
		return domain.RoleConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg domain.RoleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}
