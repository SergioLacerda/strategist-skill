package main

import (
	"fmt"
	"os"

	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
)

func installDependencies() installadapter.Dependencies {
	return installadapter.Dependencies{
		ResolveTarget:  resolveRuntimeInstallTarget,
		UserHomeDir:    os.UserHomeDir,
		ServiceFactory: newInstallService,
	}
}

func upgradeDependencies() installadapter.UpgradeDependencies {
	return installadapter.UpgradeDependencies{
		ResolveTarget:  resolveRuntimeInstallTarget,
		ServiceFactory: newUpgradeService,
	}
}

// newUpgradeService deliberately omits the Compiler, shim and awareness
// refresher that newInstallService wires: upgrade reconciles files only.
func newUpgradeService(allowDowngrade bool) installadapter.UpgradeService {
	return internalinstall.Service{
		Extractor:      embedpkg.Extractor{},
		Lister:         embedpkg.Extractor{},
		Version:        Version,
		AllowDowngrade: allowDowngrade,
	}
}

func resolveRuntimeInstallTarget(explicit string, global bool) (string, error) {
	target, err := installadapter.ResolveTarget(explicit, global, findStrategistRoot)
	if err != nil {
		return "", fmt.Errorf("resolve install target: %w", err)
	}
	return target, nil
}

func newInstallService(shimHome string) installadapter.Installer {
	return internalinstall.Service{
		Extractor:          embedpkg.Extractor{},
		Lister:             embedpkg.Extractor{},
		Compiler:           compile.Compiler{},
		ShimHomeDir:        shimHome,
		AwarenessRefresher: refreshAgentAwarenessFromEmbed,
		Version:            Version,
	}
}

func refreshAgentAwarenessFromEmbed(strategistRoot, projectRoot, version string) bool {
	tplBytes, err := embedpkg.Extractor{}.ReadFile("templates/agent-protocol.md")
	if err != nil {
		tplBytes = nil
	}
	return compile.RefreshAgentAwareness(strategistRoot, projectRoot, version, tplBytes)
}

var _ domain.FileExtractor = embedpkg.Extractor{}
