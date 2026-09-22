package install

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// refuseNormativeDowngrade stops `strategist upgrade` before any write when it
// would replace a normative file with a default this runtime already moved
// past, i.e. when the running binary is older than the runtime. Install makes
// the same decision in planRuntimeDefaultUpgrade.
func (s Service) refuseNormativeDowngrade(strategistDir string, embeddedHashes map[string]string, toWrite []string) error {
	manifest, loaded, err := loadInstallManifest(strategistDir)
	if err != nil || !loaded {
		// Without a readable manifest there is no install history to compare
		// against; PlanUpgrade and the manifest write surface real I/O errors.
		return nil //nolint:nilerr // see comment above
	}
	for _, path := range normativeOnly(toWrite) {
		if err := s.checkNormativeDowngrade(strategistDir, path, embeddedHashes[path], manifest); err != nil {
			return err
		}
	}
	return nil
}

// normativeOnly keeps the paths that are Strategist-owned normative files.
func normativeOnly(paths []string) []string {
	normative := map[string]bool{}
	for _, path := range domain.NormativeRuntimeDefaultPaths() {
		normative[path] = true
	}
	var out []string
	for _, path := range paths {
		if normative[path] {
			out = append(out, path)
		}
	}
	return out
}

func (s Service) checkNormativeDowngrade(strategistDir, path, embeddedHash string, manifest domain.InstallManifest) error {
	currentHash, exists, err := runtimefs.ReadSHA256(filepath.Join(strategistDir, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("upgrade: read normative runtime file %s: %w", path, err)
	}
	entry, ok := manifest.FileByPath(path)
	decision := domain.DecideRuntimeDefaultUpdate(domain.RuntimeDefaultDecisionInput{
		Exists: exists, CurrentHash: currentHash, EmbeddedHash: embeddedHash,
		ManifestHash: entry.SHA256, ManifestHistory: entry.History, HasManifest: ok,
		AllowDowngrade: s.AllowDowngrade,
	})
	if decision == domain.RuntimeDecisionDowngrade {
		return fmt.Errorf("upgrade: %s", domain.FormatRuntimeStaleDiagnostic(path, decision))
	}
	return nil
}
