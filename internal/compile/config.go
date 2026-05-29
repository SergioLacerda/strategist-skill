package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config reads active.yaml, personas/*.yaml and roles/*.yaml from root
// and writes a gzip-compressed JSON artifact to outputPath.
// It mirrors the logic of compile-config.sh exactly.
func Config(root, outputPath string) error {
	activePath := filepath.Join(root, "active.yaml")
	active, err := loadYAMLFile(activePath)
	if err != nil {
		return fmt.Errorf("compile config: %w", err)
	}

	sources := map[string]int64{activePath: mtime(activePath)}

	personas, err := compileYAMLDir(filepath.Join(root, "personas"), sources)
	if err != nil {
		return fmt.Errorf("compile config: personas: %w", err)
	}

	roles, err := compileYAMLDir(filepath.Join(root, "roles"), sources)
	if err != nil {
		return fmt.Errorf("compile config: roles: %w", err)
	}

	artifact := compiledConfig{
		Schema:     "strategist-compiled-config/1.0",
		CompiledAt: time.Now().Unix(),
		Sources:    sources,
		Active:     active,
		Personas:   personas,
		Roles:      roles,
	}

	if err := writeGzJSON(outputPath, artifact); err != nil {
		return fmt.Errorf("compile config: write %s: %w", outputPath, err)
	}
	return nil
}

// compileYAMLDir reads all *.yaml files from dir, adding each to sources,
// and returns a map of basename-without-ext → parsed content.
func compileYAMLDir(dir string, sources map[string]int64) (map[string]any, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	result := make(map[string]any, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, loadErr := loadYAMLFile(path)
		if loadErr != nil {
			return nil, loadErr
		}
		sources[path] = mtime(path)
		name := strings.TrimSuffix(e.Name(), ".yaml")
		result[name] = content
	}
	return result, nil
}
