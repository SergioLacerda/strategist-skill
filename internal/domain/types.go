package domain

// CompiledConfig is the gzip-compressed JSON artifact produced by compiling
// active.yaml + personas/*.yaml + roles/*.yaml.
type CompiledConfig struct {
	Schema     string           `json:"schema"`
	CompiledAt string           `json:"compiled_at"`
	Sources    map[string]int64 `json:"sources"` // path → mtime unix seconds
	Active     map[string]any   `json:"active"`
	Personas   map[string]any   `json:"personas"`
	Roles      map[string]any   `json:"roles"`
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

// InstallConfig holds parameters for the install command.
type InstallConfig struct {
	// Target is the absolute path where .strategist/ will be created.
	Target string
	Silent bool
	Wizard bool
}

// WizardConfig holds values collected from the interactive install wizard.
type WizardConfig struct {
	Mode        string
	BasePath    string
	Provider    string
	RolesConfig map[string]any
}
