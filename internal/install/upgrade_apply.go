package install

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// ApplyUpgrade writes every Missing/AutoUpgrade file (and, when force is
// true, Customized files too), snapshotting each file it is about to
// overwrite into a fresh timestamped backup dir first, then writes an
// updated full-tree install manifest. Orphaned entries are only reported —
// never deleted. Returns the backup dir (empty if nothing was overwritten).
func (s Service) ApplyUpgrade(strategistDir string, plan UpgradePlan, force bool) (backupDir string, retErr error) {
	toWrite, toBackup := upgradeWriteSet(plan, force)

	if len(toBackup) > 0 {
		var err error
		backupDir, err = s.snapshotBeforeUpgrade(strategistDir, toBackup)
		if err != nil {
			return "", fmt.Errorf("upgrade: snapshot before write: %w", err)
		}
	}

	for _, p := range toWrite {
		if err := s.writeUpgradeFile(strategistDir, p); err != nil {
			return backupDir, err
		}
	}

	fullManifest := domain.NewFullInstallManifest(packageID(s.Version), plan.embeddedHashes)
	if err := saveInstallManifest(strategistDir, fullManifest); err != nil {
		return backupDir, fmt.Errorf("upgrade: save manifest: %w", err)
	}
	return backupDir, nil
}

func upgradeWriteSet(plan UpgradePlan, force bool) (toWrite, toBackup []string) {
	for _, e := range plan.Entries {
		switch e.State {
		case domain.UpgradeMissing:
			toWrite = append(toWrite, e.Path)
		case domain.UpgradeAutoUpgrade:
			toWrite = append(toWrite, e.Path)
			toBackup = append(toBackup, e.Path)
		case domain.UpgradeCustomized:
			if force {
				toWrite = append(toWrite, e.Path)
				toBackup = append(toBackup, e.Path)
			}
		case domain.UpgradeManaged, domain.UpgradeOrphaned:
			// no-op: already current, or not ours to touch automatically.
		}
	}
	return toWrite, toBackup
}

func (s Service) writeUpgradeFile(strategistDir, relPath string) error {
	data, err := s.Extractor.ReadFile(relPath)
	if err != nil {
		return fmt.Errorf("upgrade: read embedded %s: %w", relPath, err)
	}
	target, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(relPath))
	if err != nil {
		return fmt.Errorf("upgrade: resolve %s: %w", relPath, err)
	}
	if err := runtimefs.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("upgrade: write %s: %w", relPath, err)
	}
	return nil
}

// snapshotBeforeUpgrade copies each of paths' current on-disk content into a
// new timestamped subdirectory of strategistDir/.upgrade-backups, and
// ensures that directory is gitignored. A path that no longer exists by the
// time the snapshot runs (e.g. removed between Plan and Apply) is skipped —
// there is nothing to preserve for it.
func (s Service) snapshotBeforeUpgrade(strategistDir string, paths []string) (string, error) {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	backupDir, err := runtimefs.SafeJoin(strategistDir, filepath.Join(upgradeBackupRelDir, stamp))
	if err != nil {
		return "", fmt.Errorf("resolve backup dir: %w", err)
	}

	for _, p := range paths {
		src, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(p))
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", p, err)
		}
		data, err := os.ReadFile(src) //nolint:gosec // G304: path validated by runtimefs.SafeJoin, confined to strategistDir
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("snapshot read %s: %w", p, err)
		}
		dst, err := runtimefs.SafeJoin(backupDir, filepath.FromSlash(p))
		if err != nil {
			return "", fmt.Errorf("resolve backup path %s: %w", p, err)
		}
		if err := runtimefs.WriteFile(dst, data, 0o644); err != nil {
			return "", fmt.Errorf("snapshot write %s: %w", p, err)
		}
	}

	if err := ensureGitignoreEntry(filepath.Dir(strategistDir), upgradeBackupGitignoreEntry); err != nil {
		return "", fmt.Errorf("gitignore backup dir: %w", err)
	}
	return backupDir, nil
}
