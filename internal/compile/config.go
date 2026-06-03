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

	// Validate active.yaml using a typed struct (catches required-field drift early).
	var activeCfg ActiveConfig
	if err := loadYAMLInto(activePath, &activeCfg); err != nil {
		return fmt.Errorf("compile config: %w", err)
	}
	if err := activeCfg.Validate(); err != nil {
		return fmt.Errorf("compile config: active.yaml: %w", err)
	}

	// Load as raw map so all fields (including extended ones like treasure_chests) are preserved.
	activeRaw, err := loadYAMLFile(activePath)
	if err != nil {
		return fmt.Errorf("compile config: active.yaml raw: %w", err)
	}

	sources := map[string]int64{activePath: mtime(activePath)}

	// Personas: load as raw maps to preserve all YAML fields (content_by_lang, diagnostics, etc.).
	// Validate each persona using a typed struct before including it in the artifact.
	personasRaw, err := compileYAMLDir(filepath.Join(root, "personas"), sources)
	if err != nil {
		return fmt.Errorf("compile config: personas: %w", err)
	}
	personasTyped, err := compileYAMLDirTyped[PersonaConfig](filepath.Join(root, "personas"), nil)
	if err != nil {
		return fmt.Errorf("compile config: personas validate: %w", err)
	}
	for name, persona := range personasTyped {
		if err := persona.Validate(); err != nil {
			return fmt.Errorf("compile config: personas/%s: %w", name, err)
		}
	}

	roles, err := compileYAMLDir(filepath.Join(root, "roles"), sources)
	if err != nil {
		return fmt.Errorf("compile config: roles: %w", err)
	}

	artifact := compiledConfig{
		Schema:     "strategist-compiled-config/1.0",
		CompiledAt: time.Now().Unix(),
		Sources:    sources,
		Active:     activeRaw,
		Personas:   personasRaw,
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

// compileYAMLDirTyped reads all *.yaml files from dir and returns a typed map
// keyed by basename-without-ext.
func compileYAMLDirTyped[T any](dir string, sources map[string]int64) (map[string]T, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]T{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	result := make(map[string]T, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		var content T
		if err := loadYAMLInto(path, &content); err != nil {
			return nil, err
		}
		if sources != nil {
			sources[path] = mtime(path)
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		result[name] = content
	}
	return result, nil
}
