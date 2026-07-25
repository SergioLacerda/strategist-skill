package treasure

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// YAMLWrite pairs a destination path with the YAML document to write there.
type YAMLWrite struct {
	Path string
	Doc  *yaml.Node
}

type stagedYAMLWrite struct {
	path    string
	tmpPath string
}

// WriteYAMLNodes commits every document as a single all-or-nothing batch: a prepare phase
// serializes and writes each document to a temporary sibling file, and only if every write in
// the batch prepares successfully does a commit phase rename each temporary into place. A
// prepare-phase failure leaves every destination untouched — no partial mutation reaches disk.
// A commit-phase failure (renames happen in path order) can still leave a small residual
// window where earlier files in the batch were already replaced; the returned slice and
// wrapped error report exactly which paths were committed before the failure.
func WriteYAMLNodes(writes ...YAMLWrite) ([]string, error) {
	stagedWrites, err := stageYAMLWrites(writes)
	if err != nil {
		return nil, err
	}
	return commitStagedYAMLWrites(stagedWrites)
}

func stageYAMLWrites(writes []YAMLWrite) ([]stagedYAMLWrite, error) {
	stagedWrites := make([]stagedYAMLWrite, 0, len(writes))
	for _, w := range writes {
		tmpPath, err := stageYAMLWrite(w)
		if err != nil {
			cleanupStagedYAMLWrites(stagedWrites)
			return nil, err
		}
		stagedWrites = append(stagedWrites, stagedYAMLWrite{path: w.Path, tmpPath: tmpPath})
	}
	return stagedWrites, nil
}

func stageYAMLWrite(w YAMLWrite) (string, error) {
	data, err := encodeYAMLNode(w.Doc)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", w.Path, err)
	}
	tmpPath, err := writeTempSibling(w.Path, data, 0o644)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", w.Path, err)
	}
	return tmpPath, nil
}

func encodeYAMLNode(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // matches this repo's existing YAML style
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

func cleanupStagedYAMLWrites(stagedWrites []stagedYAMLWrite) {
	for _, s := range stagedWrites {
		os.Remove(s.tmpPath) //nolint:errcheck,gosec // best-effort cleanup of temp file
	}
}

func commitStagedYAMLWrites(stagedWrites []stagedYAMLWrite) ([]string, error) {
	var written []string
	for _, s := range stagedWrites {
		if err := os.Rename(s.tmpPath, s.path); err != nil {
			return written, fmt.Errorf("write %s (already committed: %v): %w", s.path, written, err)
		}
		written = append(written, s.path)
	}
	return written, nil
}

// writeTempSibling serializes data into a temporary file in path's directory (so the later
// rename is atomic on the same filesystem) without touching path itself.
func writeTempSibling(path string, data []byte, perm fs.FileMode) (string, error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpName) //nolint:errcheck,gosec // best-effort cleanup of temp file on failure path
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close() //nolint:errcheck,gosec // best-effort close after write failure; write error is what's reported
		return "", fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close() //nolint:errcheck,gosec // best-effort close after chmod failure; chmod error is what's reported
		return "", fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp: %w", err)
	}
	cleanup = false
	return tmpName, nil
}

func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return writeFileAtomicWithRename(path, data, perm, os.Rename)
}

func writeFileAtomicWithRename(path string, data []byte, perm fs.FileMode, rename func(string, string) error) error {
	tmpName, err := writeTempSibling(path, data, perm)
	if err != nil {
		return err
	}
	if err := rename(tmpName, path); err != nil {
		os.Remove(tmpName) //nolint:errcheck,gosec // best-effort cleanup of temp file after failed rename
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
