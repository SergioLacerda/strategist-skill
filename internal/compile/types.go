package compile

// compiledConfig is the in-memory representation of a compiled config artifact.
type compiledConfig struct {
	Schema     string           `json:"schema"`
	CompiledAt int64            `json:"compiled_at"`
	Sources    map[string]int64 `json:"sources"`
	Active     map[string]any   `json:"active"`
	Personas   map[string]any   `json:"personas"`
	Roles      map[string]any   `json:"roles"`
}

// compiledDomain is the in-memory representation of a compiled domain artifact.
type compiledDomain struct {
	Schema         string                    `json:"schema"`
	CompiledAt     int64                     `json:"compiled_at"`
	Sources        map[string]int64          `json:"sources"`
	LoadAlways     map[string]any            `json:"load_always"`
	LoadByTaskType map[string]map[string]any `json:"load_by_task_type"`
}

// compiledIndex is the in-memory representation of a compiled knowledge index artifact.
type compiledIndex struct {
	Schema     string              `json:"schema"`
	CompiledAt int64               `json:"compiled_at"`
	Sources    map[string]int64    `json:"sources"`
	Tags       map[string][]string `json:"tags"`
	SourceMeta map[string]any      `json:"source_meta"`
}

// compiledManifest records artifact paths and their SHA256 checksums.
type compiledManifest struct {
	Schema      string            `json:"schema"`
	GeneratedAt int64             `json:"generated_at"`
	Artifacts   map[string]string `json:"artifacts"` // name → "sha256:<hex>"
}

// IndexSource represents one entry in knowledge.index.yaml.
type IndexSource struct {
	ID   string   `yaml:"id"`
	Tags []string `yaml:"tags"`
}

// KnowledgeIndex is the structure of knowledge.index.yaml.
type KnowledgeIndex struct {
	Sources []IndexSource `yaml:"sources"`
}

// DomainIndex is the structure of a strategist domain index.yaml.
type DomainIndex struct {
	LoadAlways     []string            `yaml:"load_always"`
	LoadByTaskType map[string][]string `yaml:"load_by_task_type"`
}

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
