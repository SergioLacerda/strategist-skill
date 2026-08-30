package install

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// upgradeBackupRelDir is where ApplyUpgrade snapshots a file before
// overwriting it, one timestamped subdirectory per upgrade run.
const upgradeBackupRelDir = ".upgrade-backups"

// UpgradePlanEntry is one file's computed upgrade state.
type UpgradePlanEntry struct {
	Path  string
	State domain.RuntimeFileUpgradeState
}

// UpgradePlan is the full-tree upgrade decision set for one .strategist/
// root, computed by PlanUpgrade without writing anything.
type UpgradePlan struct {
	Entries        []UpgradePlanEntry
	embeddedHashes map[string]string
}

// PlanUpgrade computes the upgrade state for every path in the current
// embedded default tree (via s.Lister), plus every install-manifest-tracked
// path that has dropped out of that tree (orphaned). Read-only.
func (s Service) PlanUpgrade(strategistDir string) (UpgradePlan, error) {
	if s.Lister == nil {
		return UpgradePlan{}, fmt.Errorf("upgrade: no file lister configured")
	}
	paths, err := s.Lister.AllPaths()
	if err != nil {
		return UpgradePlan{}, fmt.Errorf("upgrade: list embedded paths: %w", err)
	}

	embeddedHashes, err := s.hashEmbeddedPaths(paths)
	if err != nil {
		return UpgradePlan{}, err
	}

	manifest, manifestLoaded, err := loadInstallManifest(strategistDir)
	if err != nil {
		return UpgradePlan{}, err
	}

	entries, err := planEntriesForCurrentTree(strategistDir, paths, embeddedHashes, manifest, manifestLoaded)
	if err != nil {
		return UpgradePlan{}, err
	}
	entries = append(entries, orphanEntries(manifest, manifestLoaded, embeddedHashes)...)

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return UpgradePlan{Entries: entries, embeddedHashes: embeddedHashes}, nil
}

func (s Service) hashEmbeddedPaths(paths []string) (map[string]string, error) {
	hashes := make(map[string]string, len(paths))
	for _, p := range paths {
		data, err := s.Extractor.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("upgrade: read embedded %s: %w", p, err)
		}
		hashes[p] = domain.SHA256Hex(data)
	}
	return hashes, nil
}

func planEntriesForCurrentTree(
	strategistDir string,
	paths []string,
	embeddedHashes map[string]string,
	manifest domain.InstallManifest,
	manifestLoaded bool,
) ([]UpgradePlanEntry, error) {
	entries := make([]UpgradePlanEntry, 0, len(paths))
	for _, p := range paths {
		runtimePath, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(p))
		if err != nil {
			return nil, fmt.Errorf("upgrade: resolve %s: %w", p, err)
		}
		currentHash, exists, err := runtimefs.ReadSHA256(runtimePath)
		if err != nil {
			return nil, fmt.Errorf("upgrade: read %s: %w", p, err)
		}
		manifestFile, hasEntry := manifest.FileByPath(p)
		state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
			Exists:       exists,
			CurrentHash:  currentHash,
			EmbeddedHash: embeddedHashes[p],
			ManifestHash: manifestFile.SHA256,
			HasManifest:  manifestLoaded && hasEntry,
		})
		entries = append(entries, UpgradePlanEntry{Path: p, State: state})
	}
	return entries, nil
}

// orphanEntries reports every manifest-tracked path that is no longer part
// of the current embedded tree (embeddedHashes has no entry for it).
func orphanEntries(manifest domain.InstallManifest, manifestLoaded bool, embeddedHashes map[string]string) []UpgradePlanEntry {
	if !manifestLoaded {
		return nil
	}
	var orphans []UpgradePlanEntry
	for _, f := range manifest.Files {
		if _, stillEmbedded := embeddedHashes[f.Path]; stillEmbedded {
			continue
		}
		orphans = append(orphans, UpgradePlanEntry{Path: f.Path, State: domain.UpgradeOrphaned})
	}
	return orphans
}
