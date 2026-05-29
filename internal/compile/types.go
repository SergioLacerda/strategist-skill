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

// indexSource represents one entry in knowledge.index.yaml.
type indexSource struct {
	ID   string   `yaml:"id"`
	Tags []string `yaml:"tags"`
}

// knowledgeIndex is the structure of knowledge.index.yaml.
type knowledgeIndex struct {
	Sources []indexSource `yaml:"sources"`
}

// domainIndex is the structure of a strategist domain index.yaml.
type domainIndex struct {
	LoadAlways     []string            `yaml:"load_always"`
	LoadByTaskType map[string][]string `yaml:"load_by_task_type"`
}
