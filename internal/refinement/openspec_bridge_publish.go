package refinement

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func publish(refined, pending string, contents map[string][]byte) error {
	parent := filepath.Dir(refined)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("openspec bridge: create refined parent: %w", err)
	}
	tmp, err := stagePackage(parent, contents)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp) //nolint:errcheck // only the private staging directory
	if err := commitPackage(tmp, refined); err != nil {
		return err
	}
	return removePending(pending)
}

func stagePackage(parent string, contents map[string][]byte) (string, error) {
	tmp, err := os.MkdirTemp(parent, ".openspec-refined-")
	if err != nil {
		return "", fmt.Errorf("openspec bridge: create atomic staging directory: %w", err)
	}
	for _, name := range canonicalFiles {
		if err := os.WriteFile(filepath.Join(tmp, name), contents[name], 0o644); err != nil {
			return "", fmt.Errorf("openspec bridge: stage %s: %w", name, err)
		}
	}
	if err := requireFiles(tmp, canonicalFiles); err != nil {
		return "", fmt.Errorf("openspec bridge: validate staged package: %w", err)
	}
	return tmp, nil
}

func commitPackage(tmp, refined string) error {
	if err := os.Rename(tmp, refined); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("openspec bridge: refined package appeared during publish")
		}
		return fmt.Errorf("openspec bridge: publish refined package: %w", err)
	}
	return nil
}

func existingPackage(root string, contents map[string][]byte) (bool, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("openspec bridge: inspect existing refined package: %w", err)
	}
	if err := requireFiles(root, canonicalFiles); err != nil {
		return false, fmt.Errorf("openspec bridge: conflicting refined package: %w", err)
	}
	return comparePackage(root, contents)
}

func comparePackage(root string, contents map[string][]byte) (bool, error) {
	for _, name := range canonicalFiles {
		got, err := os.ReadFile(filepath.Join(root, name)) //nolint:gosec // root is the validated canonical package
		if err != nil {
			return false, fmt.Errorf("openspec bridge: read existing %s: %w", name, err)
		}
		if !bytes.Equal(got, contents[name]) {
			return false, fmt.Errorf("openspec bridge: existing refined package conflicts with provider change")
		}
	}
	return true, nil
}

func removePending(pending string) error {
	if err := os.Remove(pending); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("openspec bridge: remove promoted pending analysis: %w", err)
	}
	return nil
}
