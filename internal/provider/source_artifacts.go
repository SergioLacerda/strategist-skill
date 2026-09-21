package provider

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

func digestSourceFiles(files map[string][]byte) string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte(0)
		b.Write(files[key])
		b.WriteByte(0)
	}
	return digestBytes(b.Bytes())
}

func copySource(source, target string) error {
	copier := sourceCopier{source: source, target: target}
	if err := filepath.WalkDir(source, copier.visit); err != nil {
		return fmt.Errorf("copy provider source: %w", err)
	}
	return nil
}

// sourceCopier copies an operator-selected provider directory into a governed
// staging directory. It is a type, not a closure, so each step stays small.
type sourceCopier struct{ source, target string }

func (c sourceCopier) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return fmt.Errorf("walk provider source: %w", walkErr)
	}
	rel, err := filepath.Rel(c.source, path)
	if err != nil {
		return fmt.Errorf("relative provider path: %w", err)
	}
	return c.copyPath(path, rel, entry)
}

func (c sourceCopier) copyPath(path, rel string, entry fs.DirEntry) error {
	if rel == "." {
		return makeStagingDir(c.target)
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("provider source contains unsupported symlink %q", rel)
	}
	dst := filepath.Join(c.target, rel)
	if entry.IsDir() {
		return makeStagingDir(dst)
	}
	return copyProviderFile(path, dst, rel)
}

func makeStagingDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create staging directory %s: %w", dir, err)
	}
	return nil
}

func copyProviderFile(src, dst, rel string) error {
	data, err := os.ReadFile(src) //nolint:gosec // source is explicitly operator-selected
	if err != nil {
		return fmt.Errorf("read provider file %s: %w", rel, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil { //nolint:gosec // target is a governed staging directory
		return fmt.Errorf("write provider file %s: %w", rel, err)
	}
	return nil
}
