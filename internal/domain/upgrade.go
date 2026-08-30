package domain

import (
	"sort"
	"time"
)

// RuntimeFileUpgradeState classifies one runtime file's state during
// `strategist upgrade`, covering the full installed tree — unlike
// RuntimeDefaultDecision (DecideRuntimeDefaultUpdate), which only ever
// evaluates the small NormativeRuntimeDefaultFiles subset install guards
// strictly, upgrade evaluates every path the embedded defaults ship plus
// every path a prior install-manifest still remembers.
type RuntimeFileUpgradeState string

// Upgrade file states. See DecideUpgradeFileState for the decision rules.
const (
	// UpgradeManaged: on disk and already matches the current embedded default. No-op.
	UpgradeManaged RuntimeFileUpgradeState = "managed"
	// UpgradeMissing: part of current embedded defaults, absent on disk. Written.
	UpgradeMissing RuntimeFileUpgradeState = "missing"
	// UpgradeAutoUpgrade: on disk, matches a prior installed default exactly,
	// and the embedded default has since moved on. Safe to overwrite — the
	// user never touched this file, the shipped content changed under it.
	UpgradeAutoUpgrade RuntimeFileUpgradeState = "auto_upgrade"
	// UpgradeCustomized: on disk, diverges from the last hash this file was
	// installed with (or there is no prior record). Preserved unless --force.
	UpgradeCustomized RuntimeFileUpgradeState = "customized"
	// UpgradeOrphaned: tracked by a prior install manifest, no longer part of
	// the current embedded defaults (removed from the distribution). Never
	// deleted automatically — reported so the user can decide.
	UpgradeOrphaned RuntimeFileUpgradeState = "orphaned"
)

// UpgradeFileInput carries one file's hash state into DecideUpgradeFileState.
type UpgradeFileInput struct {
	Exists bool
	// CurrentHash is the on-disk file's hash. Meaningless when Exists is false.
	CurrentHash string
	// EmbeddedHash is empty when the path is not part of the current embedded
	// tree at all — the orphan signal.
	EmbeddedHash string
	// ManifestHash is the hash this path was installed with, per the prior
	// install manifest. Meaningless when HasManifest is false.
	ManifestHash string
	HasManifest  bool
}

// DecideUpgradeFileState classifies one file for `strategist upgrade`.
// Precedence: orphan check first (it does not require Exists/hash data at
// all — a path can be orphaned whether or not it still happens to exist on
// disk), then missing, then hash comparisons.
func DecideUpgradeFileState(in UpgradeFileInput) RuntimeFileUpgradeState {
	if in.EmbeddedHash == "" {
		return UpgradeOrphaned
	}
	if !in.Exists {
		return UpgradeMissing
	}
	if in.CurrentHash == in.EmbeddedHash {
		return UpgradeManaged
	}
	if in.HasManifest && in.CurrentHash == in.ManifestHash {
		return UpgradeAutoUpgrade
	}
	return UpgradeCustomized
}

// UpgradeFileWillWrite reports whether upgrade would write the embedded
// default for a file in this state (absent force, only Missing and
// AutoUpgrade write; Customized only writes when force is true).
func UpgradeFileWillWrite(state RuntimeFileUpgradeState, force bool) bool {
	switch state {
	case UpgradeMissing, UpgradeAutoUpgrade:
		return true
	case UpgradeCustomized:
		return force
	case UpgradeManaged, UpgradeOrphaned:
		return false
	default:
		return false
	}
}

// NewFullInstallManifest builds a manifest covering every path in
// pathHashes — typically the full embedded tree from FileLister.AllPaths —
// unlike NewInstallManifest, which only ever records the
// NormativeRuntimeDefaultFiles subset. `strategist upgrade` writes this
// broader manifest so a later upgrade can detect customization and orphans
// across the whole runtime tree, not just the 8 strictly-guarded files.
func NewFullInstallManifest(packageID string, pathHashes map[string]string) InstallManifest {
	paths := make([]string, 0, len(pathHashes))
	for p := range pathHashes {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	normative := make(map[string]bool, len(NormativeRuntimeDefaultFiles()))
	for _, f := range NormativeRuntimeDefaultFiles() {
		normative[f.Path] = true
	}

	files := make([]InstallManifestFile, 0, len(paths))
	for _, p := range paths {
		owner := RuntimeFileUserOwned
		if normative[p] {
			owner = RuntimeFileNormative
		}
		files = append(files, InstallManifestFile{Path: p, Owner: owner, SHA256: pathHashes[p]})
	}

	return InstallManifest{
		Schema:      "strategist.install-manifest.v1",
		PackageID:   packageID,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Files:       files,
	}
}
