package compile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// unmarshalYAML wraps yaml.Unmarshal so helpers.go does not import yaml directly.
func unmarshalYAML(data []byte, v any) error {
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("yaml: %w", err)
	}
	return nil
}
