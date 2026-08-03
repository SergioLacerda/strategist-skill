// Package validate implements the configuration checks behind the
// `strategist validate` command: active.yaml, personas/*.yaml, roles/*.yaml,
// and arbitrary YAML files.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// ActiveYAML validates a .strategist/active.yaml file.
func ActiveYAML(path string) error {
	cfg, err := readActiveConfigForValidation(path)
	if err != nil {
		return err
	}
	return validateActiveFields(cfg)
}

func readActiveConfigForValidation(path string) (domain.ActiveConfig, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: validator reads the explicit runtime file path provided by caller
	if os.IsNotExist(err) {
		return domain.ActiveConfig{}, fmt.Errorf("file not found")
	}
	if err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("read: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("invalid YAML: %w", err)
	}
	return cfg, nil
}

func validateActiveFields(cfg domain.ActiveConfig) error {
	if cfg.Mode == "" {
		return fmt.Errorf("missing required field: mode")
	}
	if cfg.BasePath == "" {
		return fmt.Errorf("missing required field: base_path")
	}
	if len(cfg.Slots) == 0 {
		return fmt.Errorf("missing required field: slots")
	}
	if cfg.Mode != "pragmatic" && cfg.Mode != "epic" {
		return fmt.Errorf("invalid mode %q (must be pragmatic or epic)", cfg.Mode)
	}
	return nil
}

// PersonasDir validates every *.yaml file under dir as a persona config.
func PersonasDir(dir string) (errs []string, checks int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("personas/: %v", err)}, 0
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		checks++
		errs = append(errs, validatePersonaFile(dir, e.Name())...)
	}
	return errs, checks
}

func validatePersonaFile(dir, name string) []string {
	path := filepath.Join(dir, name)
	raw, err := os.ReadFile(path) //nolint:gosec // G304: persona path is selected from the validated personas directory listing
	if err != nil {
		return []string{fmt.Sprintf("personas/%s: read: %v", name, err)}
	}
	var p domain.PersonaConfig
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return []string{fmt.Sprintf("personas/%s: invalid YAML: %v", name, err)}
	}
	if err := p.ValidateForRuntime(); err != nil {
		return []string{fmt.Sprintf("personas/%s: %v", name, err)}
	}
	return nil
}

// RolesDir validates every *.yaml file under dir as a role config.
func RolesDir(dir string) (errs []string, checks int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("roles/: %v", err)}, 0
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		checks++
		errs = append(errs, validateRoleFile(dir, e.Name())...)
	}
	return errs, checks
}

func validateRoleFile(dir, name string) (errs []string) {
	path := filepath.Join(dir, name)
	raw, shape, err := readRoleShape(path)
	if err != nil {
		return []string{fmt.Sprintf("roles/%s: %v", name, err)}
	}

	if _, isRoleDef := shape["role"]; isRoleDef {
		return validateRoleDefinition(name, raw)
	}

	return validateRoleSlotMap(name, raw)
}

func readRoleShape(path string) ([]byte, map[string]any, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: role path is selected from the validated roles directory listing
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}
	var shape map[string]any
	if err := yaml.Unmarshal(raw, &shape); err != nil {
		return nil, nil, fmt.Errorf("invalid YAML: %w", err)
	}
	return raw, shape, nil
}

func validateRoleDefinition(name string, raw []byte) []string {
	var role domain.RoleConfig
	if err := yaml.Unmarshal(raw, &role); err != nil {
		return []string{fmt.Sprintf("roles/%s: invalid YAML: %v", name, err)}
	}
	if err := role.Validate(); err != nil {
		return []string{fmt.Sprintf("roles/%s: %v", name, err)}
	}
	return nil
}

func validateRoleSlotMap(name string, raw []byte) []string {
	var slotMap domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &slotMap); err != nil {
		return []string{fmt.Sprintf("roles/%s: invalid YAML: %v", name, err)}
	}
	if err := slotMap.Validate(); err != nil {
		return []string{fmt.Sprintf("roles/%s: %v", name, err)}
	}
	return nil
}

// YAMLFile checks that path parses as well-formed YAML.
func YAMLFile(path string) error {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: YAML validator intentionally reads the caller-selected file
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	return nil
}
