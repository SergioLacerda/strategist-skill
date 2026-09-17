package domain

// InstallConfig holds parameters for the install command.
type InstallConfig struct {
	// Target is the absolute path where .strategist/ will be created.
	Target string
	Silent bool
	Wizard bool
	// Global installs to a global root and skips project-local behaviors
	// such as writing target/.gitignore.
	Global bool
	// Force overwrites all files, including user-modified ones.
	// When false (default), files that differ from the embedded default are preserved.
	Force bool
	// StrictCompile makes a CompileAll failure fatal (triggers rollback) instead of
	// warning-only. Default false preserves the existing warning-only behavior.
	StrictCompile bool
	// NoShim skips writing the SKILL.md shim entirely (no ~/.claude/skills write).
	NoShim bool
	// ShimPath overrides the shim destination file path. Empty means use the
	// default ~/.claude/skills/strategist/SKILL.md location. Mutually exclusive
	// with NoShim.
	ShimPath string
}

// TreasureChest is a scoped knowledge source passed to slot providers at invocation time.
// Each slot receives only the chests whose Scope includes its role.
// The skill decides how to use the chest — Strategist only passes the path and metadata.
type TreasureChest struct {
	ID          string   `yaml:"id"`
	Path        string   `yaml:"path"`
	Scope       []string `yaml:"scope"` // "all" | "discovery" | "refinement" | "execution"
	Description string   `yaml:"description,omitempty"`
}

// ProviderManifest is the structure of a provider skill manifest at
// .strategist/skills/<provider>/skill.yaml, materialized by the installer for
// default providers or placed manually for custom ones.
type ProviderManifest struct {
	ID            string `yaml:"id"`
	Status        string `yaml:"status"`
	RiskScore     string `yaml:"risk_score"`
	Category      string `yaml:"category"`
	CanonicalRole string `yaml:"canonical_role"`
}

// WizardConfig holds values collected from the interactive install wizard.
type WizardConfig struct {
	Mode               string
	BasePath           string
	UILanguage         string // en | pt-BR — wizard interface + ongoing interactions
	DocLanguage        string // en | pt-BR — generated documentation
	ChatLanguage       string // en | pt-BR — AI chat responses
	CodeLanguage       string // en | pt-BR — internal code (default: en)
	DiscoveryProvider  string // skill id for the Ranger (discovery) slot
	RefinementProvider string // skill id for the Arquivista (refinement) slot
	ExecutionProvider  string // always "sniper" — the native execution role, not a wizard-selectable governance/provider skill id
	// DiscoveryMode and RefinementMode record which pipeline the Wizard's
	// prompt resolved for that slot — SlotBindingModeRanked when the operator
	// picked the "Ranger rankeado"/certified-Ranked option, empty/Custom
	// otherwise (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
	// DEC-002). This is wizard-internal signaling only: it is never written to
	// active.yaml (DiscoveryProvider/RefinementProvider stay a plain provider
	// id either way) — only to plugins.lock's SlotBinding.Mode, the runtime
	// reinforcement DEC-002 describes.
	DiscoveryMode     string
	RefinementMode    string
	TreasureChestPath string // optional: path to a knowledge source (e.g. .sdd/source)
	// AdrCanonicalPath is the optional, project-relative destination Sniper writes ADRs
	// to instead of the <base_path>/archived/<mission_id>-adr.md fallback (see
	// contracts/narrative/07-adr.md § Canonical Destination Resolution). Empty means
	// absent — no default is invented here. Not currently collected by the interactive
	// wizard (see TestWizardDoesNotAskPermissionLevel): set this field when constructing
	// WizardConfig programmatically, or edit active.yaml by hand after install.
	AdrCanonicalPath string
	// ResolvedPluginLock is the discovery/refinement role→weapon binding
	// state resolved by the Role/Provider migration during this wizard run
	// (docs/adr/0037-wizard-role-binding-persistence.md). Zero-value
	// (empty Bindings) means the migration was not fully resolved this run
	// and nothing should be persisted — applyWizardConfig checks
	// len(Bindings) > 0 before writing plugins.lock, so a partial/failed
	// resolution never overwrites a previously good persisted binding.
	ResolvedPluginLock PluginLockFile
	// GovernancePolicy and PermissionGrants are optional operator-owned state.
	// Nil values preserve existing workspace files for legacy installs.
	GovernancePolicy *TrustPolicy
	PermissionGrants *PermissionGrantFile
}
