package compile

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// mtime returns the Unix mtime of path in seconds, or 0 on error.
func mtime(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

// writeGzJSON encodes v as JSON and writes the gzip-compressed result to outputPath.
// The parent directory is created if it does not exist.
func writeGzJSON(outputPath string, v any) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(outputPath) //nolint:gosec // G304: path derived from strategistDir, not user input
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close() //nolint:errcheck // write errors surfaced via gz.Close below

	gz := gzip.NewWriter(f)
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}
	return nil
}

// sha256Artifact returns "sha256:<hex>" for the file at path, or "unavailable" on error.
func sha256Artifact(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is a compiled artifact path
	if err != nil {
		return "unavailable"
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

// loadYAMLFile reads a YAML file and returns its content as a generic map.
// Uses the yaml.v3 package.
func loadYAMLFile(path string) (map[string]any, error) {
	var out map[string]any
	if err := loadYAMLInto(path, &out); err != nil {
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
	if err := unmarshalYAML(data, v); err != nil {
		return fmt.Errorf("parse yaml %s: %w", path, err)
	}
	return nil
}
