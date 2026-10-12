package onboarding

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"gopkg.in/yaml.v3"
)

// LoadRoleSlotMap reads the role-to-slot map shipped with the install defaults.
func LoadRoleSlotMap(extractor domain.FileExtractor) (domain.RoleSlotMap, error) {
	data, err := extractor.ReadFile(names.RoleSlotMapPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", names.RoleSlotMapPath, err)
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(data, &roleMap); err != nil {
		return nil, fmt.Errorf("%s: %w", names.RoleSlotMapPath, err)
	}
	if err := roleMap.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", names.RoleSlotMapPath, err)
	}
	return roleMap, nil
}

// LoadRoleConfig reads the configuration of one role from the install defaults.
func LoadRoleConfig(extractor domain.FileExtractor, roleName string) (domain.RoleConfig, error) {
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
