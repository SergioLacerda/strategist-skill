package support

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/filesystem"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

const gitignoreEntry = ".strategist/.compiled/"

// UpgradeBackupGitignoreEntry ignores the pre-write snapshots `strategist
// upgrade` takes before overwriting an auto_upgrade/customized(--force)
// file — these are local safety nets for `strategist upgrade --rollback`,
// not something a target repo should commit.
const UpgradeBackupGitignoreEntry = ".strategist/.upgrade-backups/"

// EnsureGitignore adds the .strategist/.compiled/ entry to target/.gitignore
// if it is not already present.
func EnsureGitignore(target string) error {
	return EnsureGitignoreEntry(target, gitignoreEntry)
}

// EnsureGitignoreEntry adds entry to target/.gitignore if not already present.
func EnsureGitignoreEntry(target, entry string) error {
	path, err := filesystem.SafeJoin(target, names.GitignoreFileName)
	if err != nil {
		return fmt.Errorf("resolve .gitignore path: %w", err)
	}

	existing, err := os.ReadFile(path) //nolint:gosec // G304: path validated by filesystem.SafeJoin, confined to target
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	if strings.Contains(string(existing), entry) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G304: path validated by filesystem.SafeJoin, confined to target
	if err != nil {
		return fmt.Errorf("open .gitignore: %w", err)
	}
	if _, err := fmt.Fprintln(f, gitignoreLine(existing, entry)); err != nil {
		return fmt.Errorf("write .gitignore: %w", errors.Join(err, f.Close()))
	}
	return CloseGitignore(f)
}

func gitignoreLine(existing []byte, entry string) string {
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		return "\n" + entry
	}
	return entry
}

// CloseGitignore closes f and wraps a close failure with the .gitignore context.
func CloseGitignore(f *os.File) error {
	if err := f.Close(); err != nil {
		return fmt.Errorf("close .gitignore: %w", err)
	}
	return nil
}
