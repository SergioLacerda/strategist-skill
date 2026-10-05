package calibration

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Dimensions of a requirement-to-task pair: each is one atomic, verifiable
// relation the provider is asked about on its own. The composition of dimensions
// into a decision is done in code, never by the provider.
const (
	DimObligation    = "obligation_preserved"
	DimKind          = "task_kind_consistent"
	DimVerification  = "verification_checks_task"
	BoundaryPairName = "requirement_pair"
)

// PairDimensions lists the dimensions in a stable order.
func PairDimensions() []string { return []string{DimKind, DimObligation, DimVerification} }

// PairRequirement is the source obligation of a pair.
type PairRequirement struct {
	Text     string `yaml:"text" json:"text"`
	Modality string `yaml:"modality" json:"modality"`
}

// PairTarget is the task that is supposed to carry the requirement.
type PairTarget struct {
	Statement    string `yaml:"statement" json:"statement"`
	DeclaredKind string `yaml:"declared_kind" json:"declared_kind"`
	Action       string `yaml:"action" json:"action"`
	Verification string `yaml:"verification" json:"verification"`
}

// Pair is one labeled requirement-to-task case. The oracle (which dimensions hold)
// and the family are for the DOJO only: they never reach the provider.
type Pair struct {
	ID          string          `yaml:"id" json:"id"`
	Parent      string          `yaml:"parent" json:"parent"`
	Family      string          `yaml:"family" json:"family"`
	Requirement PairRequirement `yaml:"requirement" json:"requirement"`
	Target      PairTarget      `yaml:"target" json:"target"`
	// Oracle maps a dimension to whether the relation holds in this pair.
	Oracle map[string]bool `yaml:"oracle" json:"oracle"`
}

// Validate rejects a pair that cannot be evaluated or that leaks its label.
func (p Pair) Validate() error {
	return errors.Join(p.validateText(), p.validateOracle())
}

func (p Pair) validateText() error {
	var problems []error
	if p.ID == "" || p.Parent == "" || p.Family == "" {
		problems = append(problems, errors.New("id, parent and family are required"))
	}
	if strings.TrimSpace(p.Requirement.Text) == "" || strings.TrimSpace(p.Target.Statement) == "" || strings.TrimSpace(p.Target.Action) == "" {
		problems = append(problems, errors.New("requirement text, target statement and target action are required"))
	}
	if leaks(p) {
		problems = append(problems, errors.New("the pair text names its own label (family, oracle or a dimension key)"))
	}
	return errors.Join(problems...)
}

func (p Pair) validateOracle() error {
	if len(p.Oracle) == 0 {
		return errors.New("an oracle with at least one dimension is required")
	}
	var problems []error
	for dim := range p.Oracle {
		if !knownDimension(dim) {
			problems = append(problems, fmt.Errorf("unknown dimension %q", dim))
		}
	}
	return errors.Join(problems...)
}

func knownDimension(dim string) bool {
	for _, known := range PairDimensions() {
		if known == dim {
			return true
		}
	}
	return false
}

// leaks reports whether text that reaches the provider contains the family name
// or a dimension key, which would hand it the answer.
func leaks(p Pair) bool {
	text := strings.ToLower(p.State())
	for _, secret := range append(PairDimensions(), strings.ToLower(p.Family)) {
		if secret != "" && strings.Contains(text, strings.ToLower(secret)) {
			return true
		}
	}
	return false
}

// State is the exact text the provider sees for the pair. The oracle, the family,
// the id and the parent are not in it.
func (p Pair) State() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## requirement\ntext: %s\nmodality: %s\n\n", strings.TrimSpace(p.Requirement.Text), strings.TrimSpace(p.Requirement.Modality))
	fmt.Fprintf(&b, "## target\nstatement: %s\ndeclared_kind: %s\naction: %s\nverification: %s\n", strings.TrimSpace(p.Target.Statement), strings.TrimSpace(p.Target.DeclaredKind), strings.TrimSpace(p.Target.Action), verificationText(p.Target.Verification))
	return b.String()
}

func verificationText(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(absent)"
	}
	return strings.TrimSpace(v)
}

// PairHeldOut is the final split, by parent so every variant of a parent stays on one side.
func PairHeldOut(parent, salt string, percent int) bool {
	return HeldOut("parent:"+parent, salt, percent)
}

// SortedPairs orders pairs by id.
func SortedPairs(pairs []Pair) []Pair {
	out := append([]Pair(nil), pairs...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// leaks keeps family names out of the provider text: the families are tokens that
// do not occur in natural text (weakened_obligation, wrong_kind, ...).
