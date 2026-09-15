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

	// nativeExecutionProvider is always the native sniper role — execution
	// has no configurable "default skill" the way discovery/refinement do
	// (see promptSlots and defaultSkillBySlot below).
	nativeExecutionProvider = "sniper"
)

// defaultSkillBySlot is the named default-skill variable per slot (DEC-004,
// mission 20260913-wizard-hardcoded-fallback-maps-cleanup), superseding the prior
// separate defaultDiscoveryProvider/defaultRefinementProvider constants.
//
// refinement's value is the embedded weapon openspec-propose, not the
// archivist native role: openspec-propose already produces a
// proposal/design/tasks package mirroring Archivist's own output contract
// (DEC-001). If the selected weapon is unavailable or invalid, the Wizard and
// strategist check fail closed; the native role is never substituted.
var defaultSkillBySlot = map[string]string{
	"discovery":  "brainstorming",
	"refinement": "openspec-propose",
}

// shimRelPath is the shim location under an agent's home config root, e.g.
// "skills/strategist/SKILL.md" under ~/.claude, ~/.gemini, or ~/.codex.
var shimRelPath = filepath.Join(installedProvidersDirName, strategistSkillName, skillMDName)

// defaultShimPath returns the default Claude shim path under homeDir:
// homeDir/.claude/skills/strategist/SKILL.md.
func defaultShimPath(homeDir string) string {
	return filepath.Join(homeDir, claudeDirName, shimRelPath)
}
