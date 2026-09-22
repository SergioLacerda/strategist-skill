package install

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// finalizeInstall compiles the runtime, refreshes agent awareness and writes the
// install manifest, carrying each file's install history forward.
func (s Service) finalizeInstall(ctx context.Context, cfg domain.InstallConfig, strategistDir string, plan runtimeDefaultPlan, fullHashes map[string]string) error {
	if err := s.compileAfterInstall(ctx, cfg, strategistDir); err != nil {
		return err
	}
	if s.AwarenessRefresher != nil {
		s.AwarenessRefresher(strategistDir, cfg.Target, s.Version)
	}
	installManifest := withInstallHistory(strategistDir, buildInstallManifest(packageID(s.Version), plan, fullHashes))
	if err := saveInstallManifest(strategistDir, installManifest); err != nil {
		return err
	}
	return nil
}
