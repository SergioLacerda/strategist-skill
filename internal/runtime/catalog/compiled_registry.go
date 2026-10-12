package catalog

import (
	"fmt"
	"reflect"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// ParseCompiledRegistryCatalog decodes the registry sections from catalog
// YAML and delegates validation to the domain.
func ParseCompiledRegistryCatalog(raw []byte) (domain.CompiledRegistry, error) {
	var document domain.CompiledRegistryDocument
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return domain.CompiledRegistry{}, fmt.Errorf("parse compiled registry catalog: %w", err)
	}
	registry, err := domain.CompiledRegistryFromDocument(document)
	if err != nil {
		return domain.CompiledRegistry{}, fmt.Errorf("validate compiled registry catalog: %w", err)
	}
	return registry, nil
}

// CompiledRegistryDrift compares only the registry sections. The provider list
// remains workspace-editable and is intentionally excluded.
func CompiledRegistryDrift(workspaceRaw, embeddedRaw []byte) (bool, error) {
	workspace, err := ParseCompiledRegistryCatalog(workspaceRaw)
	if err != nil {
		return false, fmt.Errorf("workspace catalog: %w", err)
	}
	embedded, err := ParseCompiledRegistryCatalog(embeddedRaw)
	if err != nil {
		return false, fmt.Errorf("embedded catalog: %w", err)
	}
	return !reflect.DeepEqual(workspace, embedded), nil
}
