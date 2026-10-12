package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
)

func saveInstallManifest(strategistDir string, manifest domain.InstallManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("install: marshal manifest: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(strategistDir, domain.InstallManifestRelPath)
	if err := support.AtomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("install: write manifest: %w", err)
	}
	return nil
}

func loadInstallManifest(strategistDir string) (domain.InstallManifest, bool, error) {
	path := filepath.Join(strategistDir, domain.InstallManifestRelPath)
	data, err := os.ReadFile(path) //nolint:gosec // G304: install manifest path is derived from the selected .strategist root
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.InstallManifest{}, false, nil
		}
		return domain.InstallManifest{}, false, fmt.Errorf("install: read manifest: %w", err)
	}
	var manifest domain.InstallManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return domain.InstallManifest{}, false, fmt.Errorf("install: parse manifest: %w", err)
	}
	return manifest, true, nil
}
