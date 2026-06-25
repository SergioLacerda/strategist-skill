package domain

import (
	"fmt"
	"strings"
)

// PhaseLabels holds the display name for each mission phase.
type PhaseLabels struct {
	Discovery  string `yaml:"discovery"`
	Refinement string `yaml:"refinement"`
	Execution  string `yaml:"execution"`
}

// ActiveConfig is the structure of a standalone active.yaml template.
type ActiveConfig struct {
	Mode               string `yaml:"mode"`
	BasePath           string `yaml:"base_path"`
	RolesConfig        string `yaml:"roles_config"`
	KnowledgeIndexPath string `yaml:"knowledge_index_path"`
	Language           any    `yaml:"language,omitempty"`
	AdrEnabled         bool   `yaml:"adr_enabled"`
	// Deprecated: execution_mode and git_persistence_mode are no longer written by install.
	// Kept for backwards compat — existing runtimes with these fields parse without error.
	ExecutionMode      string            `yaml:"execution_mode,omitempty"`
	GitPersistenceMode string            `yaml:"git_persistence_mode,omitempty"`
	Slots              map[string]string `yaml:"slots"`
}

// PersonaConfig is the structure of a persona yaml file (personas/*.yaml).
type PersonaConfig struct {
	ID             string      `yaml:"id"`
	Description    string      `yaml:"description"`
	PhaseLabels    PhaseLabels `yaml:"phase_labels"`
	ToneDirective  string      `yaml:"tone_directive"`
	ProgressPrefix string      `yaml:"progress_prefix"`
}

// ApprovalGateContract is the structure of contracts/approval-gate.yaml.
type ApprovalGateContract struct {
	Module      string `yaml:"module"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// CompiledConfig is the gzip-compressed JSON artifact produced by compiling
// active.yaml + personas/*.yaml + roles/*.yaml.
type CompiledConfig struct {
	Schema     string                   `json:"schema"`
	CompiledAt string                   `json:"compiled_at"`
	Sources    map[string]int64         `json:"sources"` // path → mtime unix seconds
	Active     ActiveConfig             `json:"active"`
	Personas   map[string]PersonaConfig `json:"personas"`
	Roles      map[string]any           `json:"roles"`
}

// CompiledDomain is the gzip-compressed JSON artifact produced by compiling
// the domain configuration files.
type CompiledDomain struct {
	Schema     string           `json:"schema"`
	CompiledAt string           `json:"compiled_at"`
	Sources    map[string]int64 `json:"sources"`
	Domain     map[string]any   `json:"domain"`
}

// CompiledIndex is the gzip-compressed JSON artifact produced by compiling
// knowledge.index.yaml.
type CompiledIndex struct {
	Schema     string           `json:"schema"`
	CompiledAt string           `json:"compiled_at"`
	Sources    map[string]int64 `json:"sources"`
	Index      map[string]any   `json:"index"`
}

// CompiledManifest records all artifact paths and their compile timestamps.
type CompiledManifest struct {
	Schema     string            `json:"schema"`
	CompiledAt string            `json:"compiled_at"`
	Artifacts  map[string]string `json:"artifacts"` // name → artifact path
}

// Validate returns an error if any required field is missing or invalid.
// Called after unmarshal in the bootstrap fast path to catch config drift early.
func (a ActiveConfig) Validate() error {
	var errs []string
	if a.Mode == "" {
		errs = append(errs, "mode is required")
	}
	if a.BasePath == "" {
		errs = append(errs, "base_path is required")
	}
	if a.RolesConfig == "" {
		errs = append(errs, "roles_config is required")
	}
	if len(a.Slots) == 0 {
		errs = append(errs, "slots must have at least one entry")
	}
	// Execution policy is fixed — no per-config validation needed.
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("active config invalid: %s", strings.Join(errs, "; "))
}

// Validate returns an error if any required field is missing.
func (p PersonaConfig) Validate() error {
	var errs []string
	if p.ID == "" {
		errs = append(errs, "id is required")
	}
	if p.ToneDirective == "" {
		errs = append(errs, "tone_directive is required")
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("persona config invalid: %s", strings.Join(errs, "; "))
}

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
	ID                     string `yaml:"id"`
	Status                 string `yaml:"status"`
	RiskScore              string `yaml:"risk_score"`
	Category               string `yaml:"category"`
	ProviderClass          string `yaml:"provider_class"`
	SpecializationTaxonomy struct {
		CanonicalRole string `yaml:"canonical_role"`
		ProviderClass string `yaml:"provider_class"`
	} `yaml:"specialization_taxonomy"`
}

// DojoFileCheck specifies one file that must exist and match structural requirements.
type DojoFileCheck struct {
	Path             string   `yaml:"path"`
	RequiredSections []string `yaml:"required_sections"`
	MustContain      []string `yaml:"must_contain"`
	MustNotContain   []string `yaml:"must_not_contain"`
}

// DojoPipeline specifies which slots must (or must not) be invoked and where the pipeline must stop.
type DojoPipeline struct {
	MustStopAt      string   `yaml:"must_stop_at"`
	SlotsInvoked    []string `yaml:"slots_invoked"`
	SlotsNotInvoked []string `yaml:"slots_not_invoked"`
}

// DojoEmitLog specifies OTEL emit-taxonomy keys that must and must not appear in the run log.
type DojoEmitLog struct {
	MustContain    []string `yaml:"must_contain"`
	MustNotContain []string `yaml:"must_not_contain"`
}

// DojoManifestCheck specifies a provider manifest assertion for a slot.
type DojoManifestCheck struct {
	Slot             string   `yaml:"slot"`
	ExpectedProvider string   `yaml:"expected_provider"`
	ManifestExists   bool     `yaml:"manifest_exists"`
	FieldsPresent    []string `yaml:"fields_present"`
}

// DojoTimingCriteria specifies wall-time performance constraints for a scenario.
// MaxWallTimeMs is extracted from the total_wall_time_ms= field in emit.log.
type DojoTimingCriteria struct {
	MaxWallTimeMs int `yaml:"max_wall_time_ms"`
}

// DojoCriteria is the deserialized form of a scenario's criteria.yaml.
type DojoCriteria struct {
	Scenario       string              `yaml:"scenario"`
	Description    string              `yaml:"description"`
	RunDir         string              `yaml:"run_dir"`
	AutoStopAtGate bool                `yaml:"auto_stop_at_gate"`
	FilesCreated   []DojoFileCheck     `yaml:"files_created"`
	Pipeline       DojoPipeline        `yaml:"pipeline"`
	EmitLog        DojoEmitLog         `yaml:"emit_log"`
	ManifestChecks []DojoManifestCheck `yaml:"manifest_checks"`
	TimingCriteria *DojoTimingCriteria `yaml:"timing_criteria,omitempty"`
}

// DojoCheckItem is the result of one individual check within a scenario run.
type DojoCheckItem struct {
	Label  string
	Passed bool
	Detail string
}

// DojoCheckResult is the aggregated result of running all checks for a scenario.
type DojoCheckResult struct {
	Scenario string
	Items    []DojoCheckItem
}

// Passed returns true if all items passed.
func (r DojoCheckResult) Passed() bool {
	for _, it := range r.Items {
		if !it.Passed {
			return false
		}
	}
	return true
}

// FailCount returns the number of failed items.
func (r DojoCheckResult) FailCount() int {
	n := 0
	for _, it := range r.Items {
		if !it.Passed {
			n++
		}
	}
	return n
}

// WizardConfig holds values collected from the interactive install wizard.
type WizardConfig struct {
	Mode               string
	BasePath           string
	UILanguage         string // en | pt-BR — wizard interface + ongoing interactions
	DocLanguage        string // en | pt-BR — generated documentation
	ChatLanguage       string // en | pt-BR — AI chat responses
	CodeLanguage       string // en | pt-BR — internal code (default: en)
	AdrEnabled         bool   // whether to enable the ADR opportunity stage
	DiscoveryProvider  string // skill id for the Ranger (discovery) slot
	RefinementProvider string // skill id for the Arquivista (refinement) slot
	ExecutionProvider  string // skill id for the Sniper (execution) slot
	TreasureChestPath  string // optional: path to a knowledge source (e.g. .sdd/source)
}
