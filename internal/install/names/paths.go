package names

import (
	"path/filepath"
)

// Package-local names for install-owned runtime files and embedded templates.
// Promote to domain constants only when a second package needs the same value.
const (
	StrategistDirName  = ".strategist"
	ActiveYAMLName     = "active.yaml"
	ConfigLockName     = ".config.lock"
	SkillMDName        = "SKILL.md"
	SkillYAMLName      = "skill.yaml"
	KnowledgeIndexName = "knowledge.index.yaml"
	TreasureChestsName = "treasure-chests.yaml"
	RuntimeConfigName  = "config.yaml"
	GitignoreFileName  = ".gitignore"

	EpicStandaloneTemplatePath = "templates/epic-standalone.yaml"
	KnownProvidersTemplatePath = "templates/known-providers.yaml"

	// skillsDirName is the directory that holds installed and embedded
	// skill payloads, both under an install root and under the embedded
	// defaults tree.
	SkillsDirName         = "skills"
	InternalSkillsDirName = "internal_skills"
	strategistSkillName   = "strategist"
	claudeDirName         = ".claude"

	// nativeExecutionProvider is always the native sniper role — execution
	// has no configurable "default skill" the way discovery/refinement do
	// (see promptSlots and the default skills below).
	NativeExecutionProvider = "sniper"
)

// Role ids of the native Strategist roles that Weapon bindings are certified for.
const (
	RoleRanger    = "ranger"
	RoleArchivist = "archivist"
	RoleSniper    = "sniper"
)

// Risk scores a slot may declare (domain.SlotRiskContract owns the mapping).
const (
	RiskWriteAnalysis = "write_analysis"
	RiskControlled    = "controlled"
)

// Connector identities recorded in plugin bindings and the catalog
// compatibility sources a provider can declare.
const (
	ConnectorEmbedded       = "strategist-embedded"
	ConnectorHost           = "strategist-host"
	ConnectorNativeRole     = "strategist-native-role"
	ConnectorRankedRuntime  = "strategist-ranked-runtime"
	ConnectorCurrentRuntime = "current-runtime"

	PluginBindingSchemaVersion = "strategist-plugin-binding/v1"

	CompatibilityEmbedded   = "embedded"
	CompatibilityNativeRole = "native_role"
	CompatibilityExternal   = "external"
)

// Configuration keys named in validation and prompt errors.
const (
	AdrCanonicalPathKey      = "adr.canonical_path"
	RunbooksCanonicalPathKey = "runbooks.canonical_path"
)

// Default skills of the discovery and refinement slots (DEC-004, mission
// 20260913-wizard-hardcoded-fallback-maps-cleanup). refinement's value is the
// embedded weapon openspec-propose, not the archivist native role: it already
// produces a proposal/design/tasks package mirroring Archivist's own output
// contract (DEC-001). If the selected weapon is unavailable or invalid, the
// Wizard and strategist check fail closed; the native role is never substituted.
const (
	DefaultDiscoverySkill  = "brainstorming"
	DefaultRefinementSkill = "openspec-propose"
)

// ShimRelPath is the shim location under an agent's home config root, e.g.
// "skills/strategist/SKILL.md" under ~/.claude, ~/.gemini, or ~/.codex.
var ShimRelPath = filepath.Join(SkillsDirName, strategistSkillName, SkillMDName)

// DefaultShimPath returns the default Claude shim path under homeDir:
// homeDir/.claude/skills/strategist/SKILL.md.
func DefaultShimPath(homeDir string) string {
	return filepath.Join(homeDir, claudeDirName, ShimRelPath)
}

// PluginLockFileName is the on-disk artifact persisting the resolved
// discovery/refinement role→weapon bindings across `strategist install` runs
// (docs/adr/0037-wizard-role-binding-persistence.md). Like active.yaml's
// .config.lock seal, it is machine-written/read only — never hand-edited —
// but it is a distinct concept: .config.lock is a content-hash tamper seal
// over active.yaml, this file is the actual activation record (analogous to
// domain.PluginLock/SlotBinding, not to the seal).
const PluginLockFileName = "plugins.lock"

// PluginCatalogPath is the install-root location of the plugin catalog.
const PluginCatalogPath = "plugins/catalog.yaml"

// Leveling policy files under the install root.
const (
	LevelingPolicyPath        = "leveling.yaml"
	LevelingCompatibilityPath = ".leveling-compat.yaml"
)

// RoleSlotMapPath is the install-root location of the role-to-slot map.
const RoleSlotMapPath = "roles/default.yaml"
