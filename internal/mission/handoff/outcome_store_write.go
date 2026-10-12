package handoff

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// linkExclusive writes content to a temporary file and links it to its final
// name, which fails if the name exists, so a record appears whole or not at all.
func linkExclusive(dir, name string, content []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: runtime state directory
		return fmt.Errorf("create outcome directory: %w", err)
	}
	temp, err := writeTemporary(dir, name, content)
	if err != nil {
		return err
	}
	linkErr := os.Link(temp, filepath.Join(dir, name))
	if removeErr := os.Remove(temp); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && linkErr == nil {
		return fmt.Errorf("remove temporary outcome: %w", removeErr)
	}
	if linkErr != nil {
		return fmt.Errorf("publish outcome %s: %w", name, linkErr)
	}
	return nil
}

func writeTemporary(dir, name string, content []byte) (string, error) {
	temp, err := os.CreateTemp(dir, ".tmp-"+name+"-")
	if err != nil {
		return "", fmt.Errorf("create temporary outcome: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()           //nolint:errcheck // the write error is the one worth reporting
		_ = os.Remove(temp.Name()) //nolint:errcheck // best-effort cleanup of the failed temporary
		return "", fmt.Errorf("write outcome: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(temp.Name()) //nolint:errcheck // best-effort cleanup of the failed temporary
		return "", fmt.Errorf("close outcome: %w", err)
	}
	return temp.Name(), nil
}
