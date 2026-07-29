package compile

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// mtime returns the Unix mtime of path in seconds, or 0 on error.
func mtime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

func sourceMetaForSources(sources map[string]int64) map[string]sourceMetadata {
	meta := make(map[string]sourceMetadata, len(sources))
	for path, recorded := range sources {
		info, err := os.Stat(path)
		if err != nil {
			meta[path] = sourceMetadata{MTime: recorded}
			continue
		}
		meta[path] = sourceMetadata{
			MTime:   info.ModTime().Unix(),
			MTimeNS: info.ModTime().UnixNano(),
			Size:    info.Size(),
			SHA256:  sourceFileSHA256(path),
		}
	}
	return meta
}

func sourceFileSHA256(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path comes from compile source inventory
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// writeGzJSON encodes v as JSON and writes the gzip-compressed result to outputPath.
// The parent directory is created if it does not exist.
// The write is atomic: data is written to a temp file and renamed into place so
// concurrent readers never observe a partially-written archive.
func writeGzJSON(outputPath string, v any) error {
	if err := runtimefs.WriteGzJSON(outputPath, v); err != nil {
		return fmt.Errorf("write gz json: %w", err)
	}
	return nil
}

// sha256Artifact returns "sha256:<hex>" for the file at path, or "unavailable" on error.
func sha256Artifact(path string) string {
	hash, exists, err := runtimefs.ReadSHA256(path)
	if err != nil || !exists {
		return "unavailable"
	}
	return "sha256:" + hash
}

// loadYAMLFile reads a YAML file and returns its content as a generic map.
// Uses the yaml.v3 package.
func loadYAMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	out, err := parseYAMLMapBytes(path, data)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// loadYAMLInto reads a YAML file into v.
func loadYAMLInto(path string, v any) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path derived from strategistDir
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return parseYAMLBytes(path, data, v)
}

func parseYAMLMapBytes(path string, data []byte) (map[string]any, error) {
	var out map[string]any
	if err := parseYAMLBytes(path, data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseYAMLBytes(path string, data []byte, v any) error {
	if err := unmarshalYAML(data, v); err != nil {
		return fmt.Errorf("parse yaml %s: %w", path, err)
	}
	return nil
}
