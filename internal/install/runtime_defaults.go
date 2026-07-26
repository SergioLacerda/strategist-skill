package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

type runtimeDefaultPlan struct {
	embeddedHashes map[string]string
	decisions      map[string]domain.RuntimeDefaultDecision
}

func (s Service) planRuntimeDefaultUpgrade(strategistDir string, force bool) (runtimeDefaultPlan, error) {
	embeddedHashes, err := s.embeddedNormativeHashes()
	if err != nil {
		return runtimeDefaultPlan{}, err
	}
	plan := runtimeDefaultPlan{
		embeddedHashes: embeddedHashes,
		decisions:      map[string]domain.RuntimeDefaultDecision{},
	}
	manifest, manifestLoaded, err := loadInstallManifest(strategistDir)
	if err != nil {
		return runtimeDefaultPlan{}, err
	}

	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		decision, err := planRuntimeDefaultFile(strategistDir, file.Path, embeddedHashes[file.Path], manifest, manifestLoaded, force)
		if err != nil {
			return runtimeDefaultPlan{}, err
		}
		if runtimeDefaultBlocksInstall(decision) {
			return runtimeDefaultPlan{}, fmt.Errorf("install: %s", domain.FormatRuntimeStaleDiagnostic(file.Path, decision))
		}
		plan.decisions[file.Path] = decision
	}

	return plan, nil
}

func planRuntimeDefaultFile(
	strategistDir, relPath, embeddedHash string,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	force bool,
) (domain.RuntimeDefaultDecision, error) {
	runtimePath := filepath.Join(strategistDir, filepath.FromSlash(relPath))
	currentHash, exists, readErr := runtimefs.ReadSHA256(runtimePath)
	if readErr != nil {
		return "", fmt.Errorf("install: read normative runtime file %s: %w", relPath, readErr)
	}
	manifestFile, hasManifestEntry := manifest.FileByPath(relPath)
	return domain.DecideRuntimeDefaultUpdate(domain.RuntimeDefaultDecisionInput{
		Exists:       exists,
		CurrentHash:  currentHash,
		EmbeddedHash: embeddedHash,
		ManifestHash: manifestFile.SHA256,
		HasManifest:  manifestLoaded && hasManifestEntry,
		Force:        force,
	}), nil
}

func runtimeDefaultBlocksInstall(decision domain.RuntimeDefaultDecision) bool {
	return decision == domain.RuntimeDecisionConflict || decision == domain.RuntimeDecisionUnknownManifest
}

func (s Service) embeddedNormativeHashes() (map[string]string, error) {
	hashes := make(map[string]string, len(domain.NormativeRuntimeDefaultFiles()))
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		data, err := s.Extractor.ReadFile(file.Path)
		if err != nil {
			return nil, fmt.Errorf("install: read embedded normative default %s: %w", file.Path, err)
		}
		hashes[file.Path] = domain.SHA256Hex(data)
	}
	return hashes, nil
}

func (s Service) applyRuntimeDefaultPlan(strategistDir string, plan runtimeDefaultPlan) error {
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		if err := s.applyRuntimeDefaultFile(strategistDir, file.Path, plan.decisions[file.Path]); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) applyRuntimeDefaultFile(strategistDir, relPath string, decision domain.RuntimeDefaultDecision) error {
	if decision == domain.RuntimeDecisionKeepCurrent {
		return nil
	}
	data, err := s.Extractor.ReadFile(relPath)
	if err != nil {
		return fmt.Errorf("install: read embedded normative default %s: %w", relPath, err)
	}
	targetPath := filepath.Join(strategistDir, filepath.FromSlash(relPath))
	if err := runtimefs.WriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("install: write normative default %s: %w", relPath, err)
	}
	return nil
}

func saveInstallManifest(strategistDir string, manifest domain.InstallManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("install: marshal manifest: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(strategistDir, domain.InstallManifestRelPath)
	if err := runtimefs.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("install: write manifest: %w", err)
	}
	return nil
}

func loadInstallManifest(strategistDir string) (domain.InstallManifest, bool, error) {
	path := filepath.Join(strategistDir, domain.InstallManifestRelPath)
	data, err := os.ReadFile(path)
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

func packageID(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	return version
}
