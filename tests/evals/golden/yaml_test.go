//go:build golden

package golden

import "gopkg.in/yaml.v3"

func yamlUnmarshal(data []byte, value any) error { return yaml.Unmarshal(data, value) }
