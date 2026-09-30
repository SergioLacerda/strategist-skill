package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// migrationStateFiles lists the workspace state a migrating upgrade rewrites
// besides the embedded tree: the lock and ranked runtime state when the lock
// gains versions, and the install manifest whenever files are removed or the
// lock migrates. An upgrade that only overwrites files snapshots only those.
func migrationStateFiles(strategistDir string, plan UpgradePlan, removesFiles bool) []string {
	var files []string
	if len(plan.LockMigration) > 0 {
		files = append(files, pluginLockFileName, domain.RankedRuntimeStatePath)
	}
	if removesFiles || len(plan.LockMigration) > 0 {
		files = append(files, domain.InstallManifestRelPath)
	}
	return existingRelPaths(strategistDir, files)
}

func existingRelPaths(strategistDir string, rels []string) []string {
	var existing []string
	for _, rel := range rels {
		if runtimefs.Exists(filepath.Join(strategistDir, filepath.FromSlash(rel))) {
			existing = append(existing, rel)
		}
	}
	return existing
}

// removeLegacyLayoutFiles deletes the migrated legacy files, already
// snapshotted, and prunes the directories that became empty.
func removeLegacyLayoutFiles(strategistDir string, rels []string) error {
	for _, rel := range rels {
		path, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(rel))
		if err != nil {
			return fmt.Errorf("upgrade: resolve %s: %w", rel, err)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("upgrade: remove legacy %s: %w", rel, err)
		}
		pruneEmptyDirs(strategistDir, filepath.Dir(path))
	}
	return nil
}

// pruneEmptyDirs removes dir and its empty parents up to, never including, root.
func pruneEmptyDirs(root, dir string) {
	for dir != root && strings.HasPrefix(dir, root) {
		if err := os.Remove(dir); err != nil {
			return // not empty (or already gone): stop pruning
		}
		dir = filepath.Dir(dir)
	}
}
