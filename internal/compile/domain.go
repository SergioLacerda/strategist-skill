package compile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Domain reads the domain index.yaml from root and compiles load_always
// and load_by_task_type files into a gzip-compressed JSON artifact at outputPath.
// It mirrors the logic of compile-domain.sh exactly.
func Domain(root, outputPath string) error {
	indexPath := filepath.Join(root, "index.yaml")
	indexData, err := os.ReadFile(indexPath) //nolint:gosec // G304: path derived from strategistDir root
	if err != nil {
		return fmt.Errorf("compile domain: read index.yaml: %w", err)
	}

	var idx domainIndex
	if err := yaml.Unmarshal(indexData, &idx); err != nil {
		return fmt.Errorf("compile domain: parse index.yaml: %w", err)
	}

	sources := map[string]int64{indexPath: mtime(indexPath)}

	loadAlways, err := compilePaths(root, idx.LoadAlways, sources)
	if err != nil {
		return fmt.Errorf("compile domain: load_always: %w", err)
	}

	loadByTaskType := make(map[string]map[string]any, len(idx.LoadByTaskType))
	for taskType, paths := range idx.LoadByTaskType {
		files, typeErr := compilePaths(root, paths, sources)
		if typeErr != nil {
			return fmt.Errorf("compile domain: load_by_task_type[%s]: %w", taskType, typeErr)
		}
		loadByTaskType[taskType] = files
	}

	artifact := compiledDomain{
		Schema:         "strategist-compiled-domain/1.0",
		CompiledAt:     time.Now().Unix(),
		Sources:        sources,
		LoadAlways:     loadAlways,
		LoadByTaskType: loadByTaskType,
	}

	if err := writeGzJSON(outputPath, artifact); err != nil {
		return fmt.Errorf("compile domain: write %s: %w", outputPath, err)
	}
	return nil
}

// compilePaths loads each relative path from root, records its mtime in sources,
// and returns a map of relPath → parsed YAML content.
func compilePaths(root string, relPaths []string, sources map[string]int64) (map[string]any, error) {
	result := make(map[string]any, len(relPaths))
	for _, rel := range relPaths {
		if strings.TrimSpace(rel) == "" {
			continue
		}
		full := filepath.Join(root, rel)
		content, err := loadYAMLFile(full)
		if errors.Is(err, os.ErrNotExist) {
			// mirrors compile-domain.sh behaviour: WARN and skip
			continue
		}
		if err != nil {
			return nil, err
		}
		sources[full] = mtime(full)
		result[rel] = content
	}
	return result, nil
}
