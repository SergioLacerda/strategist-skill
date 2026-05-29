package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const gitignoreEntry = ".strategist/.compiled/"

// ensureGitignore adds the .strategist/.compiled/ entry to target/.gitignore
// if it is not already present.
func ensureGitignore(target string) error {
	path := filepath.Join(target, ".gitignore")

	existing, err := os.ReadFile(path) //nolint:gosec // G304: path is constructed from cfg.Target, not user input
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	if strings.Contains(string(existing), gitignoreEntry) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G304: path is constructed from cfg.Target
	if err != nil {
		return fmt.Errorf("open .gitignore: %w", err)
	}
	defer f.Close() //nolint:errcheck

	line := gitignoreEntry
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		line = "\n" + line
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
