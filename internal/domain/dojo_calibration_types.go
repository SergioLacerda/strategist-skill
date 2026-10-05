package domain

// KindJudgeCalibration is the DOJO scenario kind that evaluates a judge against
// labeled cases instead of checking pipeline files and emit events.
const KindJudgeCalibration = "judge_calibration"

// DojoRiskPolicy is the explicit risk policy of a calibration case.
type DojoRiskPolicy struct {
	MaxFalseApproval  float64 `yaml:"max_false_approval"`
	MaxFalseRejection float64 `yaml:"max_false_rejection"`
}

// DojoCorpus selects the packages a case draws its cases from.
type DojoCorpus struct {
	// Roots are directories relative to the workspace base path. Empty means the
	// boundary default (refined and done for packages, pending for Ranger artifacts).
	Roots []string `yaml:"roots,omitempty"`
	// Include restricts the corpus to these package names; empty means all.
	Include []string `yaml:"include,omitempty"`
	// HeldOutPercent is the share (0 to 90) of packages held out of threshold selection.
	HeldOutPercent int `yaml:"held_out_percent,omitempty"`
	// Salt varies the deterministic held-out split.
	Salt string `yaml:"salt,omitempty"`
	// Synthetic marks the roots as synthetic fixtures: they are golden by
	// construction (no promotion needed) and contain nothing of any mission, so the
	// consent term names them as such. Roots must be under dojo/.
	Synthetic bool `yaml:"synthetic,omitempty"`
}

// DojoCalibration is the calibration block of a judge_calibration criteria.yaml.
type DojoCalibration struct {
	// Layer is "judge" (the real provider), "integration" (the approval policy against
	// scripted answers, no calls), "live" (the provider smoke check, synthetic text only)
	// or "pair" (atomic requirement-to-task questions, one dimension at a time).
	Layer string `yaml:"layer,omitempty"`
	// Boundary is "package" or "ranger".
	Boundary string `yaml:"boundary"`
	// Profile is the question profile asked of the provider: handoff-v1 (default), handoff-v2 or
	// handoff-atomic-v1. It is independent of the provider's configured profile, so a variant
	// is measured before it is adopted.
	Profile string `yaml:"profile,omitempty"`
	// Provider names the system under evaluation (default jev).
	Provider  string     `yaml:"provider,omitempty"`
	Corpus    DojoCorpus `yaml:"corpus,omitempty"`
	Operators []string   `yaml:"operators,omitempty"`
	// LocalScope names operators whose protection is delegated to the local checks:
	// the judge gates do not read them and a local coverage gate must pass for them.
	LocalScope []string `yaml:"local_scope,omitempty"`
	// Pairs is the pair fixtures file of a pair-layer case, relative to the case directory (default pairs.yaml).
	Pairs string `yaml:"pairs,omitempty"`
	// GateRule is gates-1 (the rule the first pilots ran under) gates-2 (default) or gates-3 (cross-validated, pre-registered for the real corpus).
	GateRule    string         `yaml:"gate_rule,omitempty"`
	Stage       int            `yaml:"stage,omitempty"`
	Seed        int64          `yaml:"seed,omitempty"`
	Repetitions int            `yaml:"repetitions,omitempty"`
	Budget      int            `yaml:"budget,omitempty"`
	RiskPolicy  DojoRiskPolicy `yaml:"risk_policy"`
	// Output is the case's own output directory relative to the workspace base
	// path. Empty means <base_path>/dojo/<case>/.
	Output string `yaml:"output,omitempty"`
	// MaxAgeDays bounds how old the case's calibration may be when pinned.
	MaxAgeDays int `yaml:"max_age_days,omitempty"`
}

// DojoCalibrationResult is the optional calibration section of a check result.
type DojoCalibrationResult struct {
	Status                string   `json:"status"`
	FailingGate           string   `json:"failing_gate,omitempty"`
	Threshold             *float64 `json:"threshold,omitempty"`
	RecordID              string   `json:"record_id,omitempty"`
	Golden                int      `json:"golden_positives"`
	Defects               int      `json:"defect_negatives"`
	CurrentFalseApproval  float64  `json:"current_false_approval"`
	CurrentFalseRejection float64  `json:"current_false_rejection"`
	CurrentThreshold      float64  `json:"current_threshold"`
	Model                 string   `json:"model,omitempty"`
	Profile               string   `json:"profile,omitempty"`
}
