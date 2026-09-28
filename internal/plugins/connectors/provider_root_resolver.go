package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ProviderRoot identifies one host runtime root. The root itself is not a
// provider and must contain a skills/<id> directory.
type ProviderRoot struct {
	Path   string
	Origin string
}

// Provider origins record which skill root a Custom provider was found in.
const (
	ProviderOriginWorkspaceAgents = "workspace:.agents"
	ProviderOriginWorkspaceCodex  = "workspace:.codex"
	ProviderOriginGlobalAgents    = "global:.agents"
	ProviderOriginGlobalCodex     = "global:.codex"
)

// ResolvedProviderPackage contains structural evidence for a selected Custom
// provider. It deliberately does not claim that the package is invocable.
type ResolvedProviderPackage struct {
	Package    domain.PluginPackage
	Path       string
	Origin     string
	SeedPath   string
	Entrypoint string
}

// ResolveCustomProviderPackage resolves a Custom provider local-first. A
// present but malformed local candidate is a hard failure: falling through to
// a global package would hide workspace drift and silently change behavior.
func ResolveCustomProviderPackage(workspaceRoot, providerID string, globalRoots []ProviderRoot) (ResolvedProviderPackage, error) {
	if strings.TrimSpace(providerID) == "" {
		return ResolvedProviderPackage{}, fmt.Errorf("custom provider id is empty")
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		return ResolvedProviderPackage{}, fmt.Errorf("custom provider workspace root is empty")
	}
	roots := append([]ProviderRoot{
		{Path: filepath.Join(workspaceRoot, ".agents"), Origin: ProviderOriginWorkspaceAgents},
		{Path: filepath.Join(workspaceRoot, ".codex"), Origin: ProviderOriginWorkspaceCodex},
	}, globalRoots...)

	for _, root := range roots {
		resolved, found, err := resolveProviderInRoot(root, providerID)
		if err != nil {
			return ResolvedProviderPackage{}, err
		}
		if found {
			return resolved, nil
		}
	}
	return ResolvedProviderPackage{}, fmt.Errorf("custom provider %q was not found in local or global skill roots", providerID)
}

// resolveProviderInRoot reports whether root holds providerID. A present but
// malformed candidate is an error, never a miss.
func resolveProviderInRoot(root ProviderRoot, providerID string) (ResolvedProviderPackage, bool, error) {
	candidate := filepath.Join(root.Path, "skills", providerID)
	present, err := pathPresent(candidate)
	if err != nil {
		return ResolvedProviderPackage{}, false, fmt.Errorf("inspect %s provider %q: %w", root.Origin, providerID, err)
	}
	if !present {
		return ResolvedProviderPackage{}, false, nil
	}
	pkg, err := ResolveLocalPackage(candidate)
	if err != nil {
		return ResolvedProviderPackage{}, false, fmt.Errorf("custom provider %q at %s is invalid: %w", providerID, candidate, err)
	}
	if pkg.ID != providerID {
		return ResolvedProviderPackage{}, false, fmt.Errorf("custom provider %q at %s declares package id %q", providerID, candidate, pkg.ID)
	}
	return ResolvedProviderPackage{
		Package: pkg, Path: candidate, Origin: root.Origin,
		SeedPath: candidate, Entrypoint: filepath.Join(candidate, requiredSkillManifestFile),
	}, true, nil
}

// DefaultGlobalProviderRoots derives global roots from an explicit home path.
// Keeping this function parameterized makes callers and tests independent of
// the developer machine and avoids hardcoded user paths.
func DefaultGlobalProviderRoots(homeDir string) []ProviderRoot {
	if strings.TrimSpace(homeDir) == "" {
		return nil
	}
	return []ProviderRoot{
		{Path: filepath.Join(homeDir, ".agents"), Origin: ProviderOriginGlobalAgents},
		{Path: filepath.Join(homeDir, ".codex"), Origin: ProviderOriginGlobalCodex},
	}
}

func pathPresent(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return true, fmt.Errorf("candidate is not a directory")
	}
	return true, nil
}
