package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Index reads knowledge.index.yaml and produces a gzip-compressed JSON
// artifact at outputPath containing an inverted tag index.
// It mirrors the logic of compile-knowledge-index.sh exactly.
func Index(knowledgeIndexPath, outputPath string) error {
	data, err := os.ReadFile(knowledgeIndexPath) //nolint:gosec // G304: path derived from strategistDir root
	if err != nil {
		return fmt.Errorf("compile index: read %s: %w", knowledgeIndexPath, err)
	}

	var ki knowledgeIndex
	if err := yaml.Unmarshal(data, &ki); err != nil {
		return fmt.Errorf("compile index: parse yaml: %w", err)
	}

	absSource, err := filepath.Abs(knowledgeIndexPath)
	if err != nil {
		return fmt.Errorf("compile index: abs path: %w", err)
	}

	// Build inverted tag index: tag → []sourceID
	tagIndex := make(map[string][]string)
	// Build source_meta: id → full source object
	sourceMeta := make(map[string]any, len(ki.Sources))

	for _, src := range ki.Sources {
		sourceMeta[src.ID] = src
		for _, tag := range src.Tags {
			tagIndex[tag] = append(tagIndex[tag], src.ID)
		}
	}

	artifact := compiledIndex{
		Schema:     "strategist-compiled-index/1.0",
		CompiledAt: time.Now().Unix(),
		Sources:    map[string]int64{absSource: mtime(knowledgeIndexPath)},
		Tags:       tagIndex,
		SourceMeta: sourceMeta,
	}

	if err := writeGzJSON(outputPath, artifact); err != nil {
		return fmt.Errorf("compile index: write %s: %w", outputPath, err)
	}
	return nil
}
