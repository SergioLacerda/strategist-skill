package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
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
	Entries []UpgradePlanEntry
	// LockMigration names the slots whose persisted Ranked binding has no
	// weapon_version yet; applying the upgrade fills it from the certified
	// binding (ADR-0061 Decision 10). Empty when nothing needs migrating.
	LockMigration  []string
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

	entries, err := s.planCurrentTreeWithRegistryRefresh(strategistDir, paths, embeddedHashes, manifest, manifestLoaded)
	if err != nil {
		return UpgradePlan{}, err
	}
	orphans, err := orphanEntries(strategistDir, manifest, manifestLoaded, embeddedHashes)
	if err != nil {
		return UpgradePlan{}, err
	}
	entries = append(entries, orphans...)
	lockMigration, err := lockSlotsNeedingMigration(strategistDir)
	if err != nil {
		return UpgradePlan{}, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return UpgradePlan{Entries: entries, LockMigration: lockMigration, embeddedHashes: embeddedHashes}, nil
}

// refreshDriftedRegistryCatalog promotes a locally edited plugins/catalog.yaml
// to an automatic (backed-up) upgrade when its compiled registry differs from
// the one built into this binary. The registry is binary-owned: mission invoke
// refuses a divergent workspace copy, so "preserve customizations" would leave
// the workspace unusable. Catalogs that only differ outside the registry
// sections stay customized and are preserved.
func (s Service) planCurrentTreeWithRegistryRefresh(strategistDir string, paths []string, embeddedHashes map[string]string, manifest domain.InstallManifest, manifestLoaded bool) ([]UpgradePlanEntry, error) {
	entries, err := planEntriesForCurrentTree(strategistDir, paths, embeddedHashes, manifest, manifestLoaded)
	if err != nil {
		return nil, err
	}
	if err := s.refreshCatalogEntry(strategistDir, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// refreshCatalogEntry promotes a customized catalog entry to an automatic
// (backed-up) upgrade when its registry drifted from the binary.
func (s Service) refreshCatalogEntry(strategistDir string, entries []UpgradePlanEntry) error {
	i := customizedCatalogIndex(entries)
	if i < 0 {
		return nil
	}
	drifted, err := s.catalogRegistryDrifted(strategistDir)
	if err != nil || !drifted {
		return err
	}
	entries[i].State = domain.UpgradeAutoUpgrade
	return nil
}

func customizedCatalogIndex(entries []UpgradePlanEntry) int {
	for i, entry := range entries {
		if entry.Path == pluginCatalogPath && entry.State == domain.UpgradeCustomized {
			return i
		}
	}
	return -1
}

// catalogRegistryDrifted reports whether the workspace catalog's registry
// differs from (or cannot be parsed against) the embedded one.
func (s Service) catalogRegistryDrifted(strategistDir string) (bool, error) {
	runtimePath, err := runtimefs.SafeJoin(strategistDir, filepath.FromSlash(pluginCatalogPath))
	if err != nil {
		return false, fmt.Errorf("upgrade: resolve %s: %w", pluginCatalogPath, err)
	}
	workspaceRaw, err := os.ReadFile(runtimePath) //nolint:gosec // G304: SafeJoin-resolved path under the runtime root
	if err != nil {
		return false, fmt.Errorf("upgrade: read %s: %w", pluginCatalogPath, err)
	}
	embeddedRaw, err := s.Extractor.ReadFile(pluginCatalogPath)
	if err != nil {
		return false, fmt.Errorf("upgrade: read embedded %s: %w", pluginCatalogPath, err)
	}
	drifted, err := catalog.CompiledRegistryDrift(workspaceRaw, embeddedRaw)
	return drifted || err != nil, nil // an unparseable workspace catalog cannot be trusted either
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
