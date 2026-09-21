package leveling

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Level sources recorded next to every resolved level, in precedence order.
const (
	SourceManual = "manual"
	SourceHost   = "host"
	SourcePolicy = "policy"
)

// Host carries the model and effort the host reports for the running agent.
type Host struct {
	Model  string
	Effort string
}

// Level is the model and effort a role runs at, plus where the value came from.
type Level struct {
	Role   string `json:"role" yaml:"role"`
	Model  string `json:"model" yaml:"model"`
	Effort string `json:"effort" yaml:"effort"`
	Source string `json:"level_source" yaml:"level_source"`
}

// DisplayName returns the short model name for a provider model id: the
// configured display name, otherwise the id without its `<provider>-` prefix
// and with its first letter upper-cased.
func (p Policy) DisplayName(provider, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if profile, ok := p.Providers[strings.ToUpper(strings.TrimSpace(provider))]; ok {
		if name := strings.TrimSpace(profile.Display[model]); name != "" {
			return name
		}
	}
	prefix := strings.ToLower(strings.TrimSpace(provider)) + "-"
	if len(model) > len(prefix) && strings.EqualFold(model[:len(prefix)], prefix) {
		model = model[len(prefix):]
	}
	return capitalize(model)
}

// Manual carries the operator's choice for a role from active.yaml.
type Manual struct {
	Model  string
	Effort string
}

// PolicyLoader loads the LEVELING policy. It is called at most once, and only
// when a value is still missing after the cheaper sources.
type PolicyLoader func() (Policy, error)

// ResolveLevel resolves the level of a role from an already loaded policy.
// Host-reported values win over the policy suggestion; a partial host report is
// completed from the policy when a provider is given. Without a provider or host
// data the level is unknown.
func ResolveLevel(policy Policy, provider, role string, signals Signals, host Host) (Level, error) {
	return ResolveLevelLazy(func() (Policy, error) { return policy, nil }, provider, role, signals, Manual{}, host)
}

// ResolveLevelLazy resolves the level of a role field by field with the
// precedence manual, then host, then policy. Level.Source names the
// highest-precedence source that contributed. The policy is loaded through load
// only when model or effort is still missing and a provider is given, so a
// complete manual choice or a complete host report never reads LEVELING data.
func ResolveLevelLazy(load PolicyLoader, provider, role string, signals Signals, manual Manual, host Host) (Level, error) {
	level := Level{Role: strings.ToLower(strings.TrimSpace(role))}
	fillLevelSource(&level, SourceManual, manual.Model, manual.Effort)
	fillLevelSource(&level, SourceHost, host.Model, host.Effort)
	if strings.TrimSpace(provider) == "" || (level.Model != "" && level.Effort != "") {
		return level, nil
	}
	return completeLevelFromPolicy(load, level, provider, role, signals)
}

func fillLevelSource(level *Level, source, model, effort string) {
	model, effort = capitalize(strings.TrimSpace(model)), strings.ToLower(strings.TrimSpace(effort))
	if ((level.Model == "" && model != "") || (level.Effort == "" && effort != "")) && level.Source == "" {
		level.Source = source
	}
	if level.Model == "" {
		level.Model = model
	}
	if level.Effort == "" {
		level.Effort = effort
	}
}

func completeLevelFromPolicy(load PolicyLoader, level Level, provider, role string, signals Signals) (Level, error) {
	policy, err := load()
	if err != nil {
		return Level{}, err
	}
	suggestion, err := Suggest(policy, provider, role, signals)
	if err != nil {
		return Level{}, err
	}
	if level.Source == "" {
		level.Source = SourcePolicy
	}
	if level.Model == "" {
		level.Model = policy.DisplayName(suggestion.Provider, suggestion.Model)
	}
	if level.Effort == "" {
		level.Effort = suggestion.Effort
	}
	return level, nil
}

// Unknown reports whether no model or effort is known.
func (l Level) Unknown() bool { return l.Model == "" && l.Effort == "" }

// Label renders `Model-Effort`, or an empty string when the level is unknown.
func (l Level) Label() string {
	switch {
	case l.Model != "" && l.Effort != "":
		return l.Model + "-" + capitalize(l.Effort)
	case l.Model != "":
		return l.Model
	default:
		return capitalize(l.Effort)
	}
}

// Tag renders the inline form `(Model-Effort)` used right after a role name in a
// narration line, or an empty string when the level is unknown.
func (l Level) Tag() string {
	if l.Unknown() {
		return ""
	}
	return "(" + l.Label() + ")"
}

// Render formats one role message. When width cannot be measured (width <= 0)
// or the inline form does not fit, the stacked layout is used:
//
//	Fase: 01/04
//	Ranger
//	Sonnet-High
//	<message>
//
// Otherwise it renders `Ranger(Sonnet-High) - <message>`. An unknown level
// keeps the unlabelled `Role - <message>` form.
func Render(level Level, message string, width int) string {
	return RenderWith(domain.DefaultRoleRegistry(), level, message, width)
}

// RenderWith is Render with an explicit role registry, which supplies the phase
// counter of each role and its derived total. Roles the registry does not know
// (for example transport) omit the counter.
func RenderWith(reg domain.RoleRegistry, level Level, message string, width int) string {
	name := capitalize(level.Role)
	if level.Unknown() {
		return fmt.Sprintf("%s - %s", name, message)
	}
	inline := fmt.Sprintf("%s(%s) - %s", name, level.Label(), message)
	if width > 0 && !strings.Contains(message, "\n") && utf8.RuneCountInString(inline) <= width {
		return inline
	}
	lines := make([]string, 0, 4)
	if phase, ok := reg.PhaseOf(level.Role); ok {
		lines = append(lines, fmt.Sprintf("Fase: %02d/%02d", phase, reg.PhaseTotal()))
	}
	lines = append(lines, name, level.Label(), message)
	return strings.Join(lines, "\n")
}

func capitalize(value string) string {
	if value == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(value)
	return strings.ToUpper(string(r)) + value[size:]
}
