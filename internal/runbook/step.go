package runbook

import "errors"

// Level is a Check's enforcement strength, per design.md § Sidecar Schema.
type Level string

// Check levels. A mandatory check blocks ValidateCompletion unless
// explicitly justified; the others never block, they only inform selection
// and review.
const (
	LevelMandatory     Level = "mandatory"
	LevelRecommended   Level = "recommended"
	LevelConditional   Level = "conditional"
	LevelInformational Level = "informational"
)

var allowedLevels = stringSet(string(LevelMandatory), string(LevelRecommended), string(LevelConditional), string(LevelInformational))

// Check is one leveled item a runbook expects to hold before/while it is
// considered complete.
type Check struct {
	ID    string `yaml:"id"`
	Level Level  `yaml:"level"`
	// When is a condition string, meaningful only when Level is
	// LevelConditional — see design.md § Sidecar Schema.
	When string `yaml:"when,omitempty"`
}

// ValidateCheck checks a Check against its required fields and allowed
// values, and that When is only set for LevelConditional checks (the field
// has no meaning otherwise).
func ValidateCheck(c Check) error {
	var errs []error
	errs = append(errs, validateNamedValue("check_invalid", "id", c.ID, nil)...)
	errs = append(errs, validateNamedValue("check_invalid", "level", string(c.Level), allowedLevels)...)
	if c.When != "" && c.Level != LevelConditional {
		errs = append(errs, errors.New("check_invalid: when is only meaningful when level is conditional"))
	}
	return errors.Join(errs...)
}
