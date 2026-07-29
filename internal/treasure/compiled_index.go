package treasure

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadCompiledIndex reads the compiled fast-path index and returns source ids and timestamp.
func LoadCompiledIndex(root string) (present map[string]bool, compiledAt int64, err error) {
	path := filepath.Join(root, ".compiled", ".index.gz")
	idx, ok, err := readCompiledIndex(path)
	if err != nil || !ok {
		return nil, 0, err
	}
	return compiledSourcePresence(idx.SourceMeta), idx.CompiledAt, nil
}

type compiledIndexYAML struct {
	CompiledAt int64          `json:"compiled_at"`
	SourceMeta map[string]any `json:"source_meta"`
}

func readCompiledIndex(path string) (compiledIndexYAML, bool, error) {
	f, err := os.Open(path) //nolint:gosec // G304
	if os.IsNotExist(err) {
		return compiledIndexYAML{}, false, nil
	}
	if err != nil {
		return compiledIndexYAML{}, false, fmt.Errorf("open .compiled/.index.gz: %w", err)
	}
	defer closeCompiledFile(f, &err)

	gz, err := gzip.NewReader(f)
	if err != nil {
		return compiledIndexYAML{}, false, fmt.Errorf("decompress .compiled/.index.gz: %w", err)
	}
	defer closeCompiledGzip(gz, &err)

	var idx compiledIndexYAML
	if err := json.NewDecoder(gz).Decode(&idx); err != nil {
		return compiledIndexYAML{}, false, fmt.Errorf("decode .compiled/.index.gz: %w", err)
	}
	return idx, true, nil
}

func closeCompiledFile(f *os.File, err *error) {
	if closeErr := f.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("close .compiled/.index.gz: %w", closeErr)
	}
}

func closeCompiledGzip(gz *gzip.Reader, err *error) {
	if closeErr := gz.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("close decompressed .compiled/.index.gz: %w", closeErr)
	}
}

func compiledSourcePresence(sourceMeta map[string]any) map[string]bool {
	present := make(map[string]bool, len(sourceMeta))
	for id := range sourceMeta {
		present[id] = true
	}
	return present
}
