package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// ListUpgradeBackups returns the available backup stamps under
// strategistDir/.upgrade-backups, newest first — each one a directory
// snapshotBeforeUpgrade wrote before an upgrade run overwrote files.
func ListUpgradeBackups(strategistDir string) ([]string, error) {
	root, err := runtimefs.SafeJoin(strategistDir, upgradeBackupRelDir)
	if err != nil {
		return nil, fmt.Errorf("resolve backup dir: %w", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list backups: %w", err)
	}
	stamps := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			stamps = append(stamps, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(stamps)))
	return stamps, nil
}

// RollbackUpgrade restores every file found in the given backup stamp
// (as produced by ListUpgradeBackups) back onto strategistDir, overwriting
// whatever `strategist upgrade` wrote in that run. It does not touch files
// the backup does not contain, and it does not delete the backup itself —
// a rollback can be repeated safely.
func RollbackUpgrade(strategistDir, stamp string) (restoredCount int, retErr error) {
	backupDir, err := runtimefs.SafeJoin(strategistDir, filepath.Join(upgradeBackupRelDir, stamp))
	if err != nil {
		return 0, fmt.Errorf("resolve backup %s: %w", stamp, err)
	}
	if !runtimefs.Exists(backupDir) {
		return 0, fmt.Errorf("upgrade: no backup found for %q under %s", stamp, upgradeBackupRelDir)
	}

	count := 0
	walkErr := filepath.WalkDir(backupDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(backupDir, path)
		if err != nil {
			return fmt.Errorf("relativize %s: %w", path, err)
		}
		data, err := os.ReadFile(path) //nolint:gosec // G304: path enumerated by WalkDir under runtimefs.SafeJoin-validated backupDir, not external input
		if err != nil {
			return fmt.Errorf("read backup %s: %w", rel, err)
		}
		dst, err := runtimefs.SafeJoin(strategistDir, rel)
		if err != nil {
			return fmt.Errorf("resolve restore target %s: %w", rel, err)
		}
		if err := runtimefs.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("restore %s: %w", rel, err)
		}
		count++
		return nil
	})
	if walkErr != nil {
		return count, fmt.Errorf("upgrade: rollback %s: %w", stamp, walkErr)
	}
	return count, nil
}
