package install

import "path/filepath"

// Package-local names for install-owned runtime files and embedded templates.
// Promote to domain constants only when a second package needs the same value.
const (
	strategistDirName  = ".strategist"
	activeYAMLName     = "active.yaml"
	configLockName     = ".config.lock"
	skillMDName        = "SKILL.md"
	skillYAMLName      = "skill.yaml"
	knowledgeIndexName = "knowledge.index.yaml"
	treasureChestsName = "treasure-chests.yaml"

	epicStandaloneTemplatePath = "templates/epic-standalone.yaml"
	knownProvidersTemplatePath = "templates/known-providers.yaml"

	installedProvidersDirName = "skills"
	strategistSkillName       = "strategist"
	claudeDirName             = ".claude"

	defaultDiscoveryProvider  = "brainstorming"
	defaultRefinementProvider = "openspec-explore"
	nativeExecutionProvider   = "sniper"
)

// shimRelPath is the shim location under an agent's home config root, e.g.
// "skills/strategist/SKILL.md" under ~/.claude, ~/.gemini, or ~/.codex.
var shimRelPath = filepath.Join(installedProvidersDirName, strategistSkillName, skillMDName)

// defaultShimPath returns the default Claude shim path under homeDir:
// homeDir/.claude/skills/strategist/SKILL.md.
func defaultShimPath(homeDir string) string {
	return filepath.Join(homeDir, claudeDirName, shimRelPath)
}
