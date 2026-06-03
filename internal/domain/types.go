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
	Mode               string            `yaml:"mode"`
	BasePath           string            `yaml:"base_path"`
	RolesConfig        string            `yaml:"roles_config"`
	KnowledgeIndexPath string            `yaml:"knowledge_index_path"`
	Language           any               `yaml:"language,omitempty"`
	AdrEnabled         bool              `yaml:"adr_enabled"`
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

// WizardConfig holds values collected from the interactive install wizard.
type WizardConfig struct {
	Mode               string
	BasePath           string
	MissionMode        string // analise | entrega_revisada | entrega_executada
	DoneScope          string // analise | entrega
	ApplyChanges       bool   // false by default; forced false when DoneScope=analise
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
