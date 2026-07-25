package domain

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
