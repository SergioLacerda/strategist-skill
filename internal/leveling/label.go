package leveling

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Level sources recorded next to every resolved level, in precedence order.
const (
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
	Role           string `json:"role" yaml:"role"`
	Model          string `json:"model" yaml:"model"`
	Effort         string `json:"effort" yaml:"effort"`
	Source         string `json:"level_source" yaml:"level_source"`
	Provider       string `json:"provider,omitempty" yaml:"provider,omitempty"`
	ModelSource    string `json:"model_source,omitempty" yaml:"model_source,omitempty"`
	EffortSource   string `json:"effort_source,omitempty" yaml:"effort_source,omitempty"`
	Capability     string `json:"capability,omitempty" yaml:"capability,omitempty"`
	FallbackUsed   bool   `json:"fallback_used,omitempty" yaml:"fallback_used,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty" yaml:"fallback_reason,omitempty"`
	PolicyVersion  int    `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest   string `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
}

// PolicyLoader loads the LEVELING policy. It is called at most once, and only
// when a value is still missing after the cheaper sources.
type PolicyLoader func() (Policy, error)

// ResolveLevel resolves the level of a role from an already loaded policy.
// Host-reported values win over the policy suggestion; a partial host report is
// completed from the policy when a provider is given. Without a provider or host
// data the level is unknown.
func ResolveLevel(policy Policy, provider, role string, signals Signals, host Host) (Level, error) {
	return ResolveLevelLazy(func() (Policy, error) { return policy, nil }, provider, role, signals, host)
}

// ResolveLevelLazy resolves the level of a role field by field with the
// precedence host, then policy. Level.Source names the highest-precedence
// source that contributed. The policy is loaded through load only when model or
// effort is still missing and a provider is given, so a complete host report,
// or an empty provider (manual mode: host passthrough), never reads LEVELING data.
func ResolveLevelLazy(load PolicyLoader, provider, role string, signals Signals, host Host) (Level, error) {
	level := Level{Role: NormalizeRole(role)}
	fillLevelSource(&level, SourceHost, host.Model, host.Effort)
	if strings.TrimSpace(provider) == "" || (level.Model != "" && level.Effort != "") {
		// No policy is loaded on this path, so the model id can only be
		// shortened by the policy-free fallback.
		level.Model = displayFallback(provider, level.Model)
		return level, nil
	}
	return completeLevelFromPolicy(load, level, provider, role, signals)
}

// fillLevelSource records a source's model and effort. The model is kept as the
// raw id so a later policy load can still match it against the configured
// display names; shortening happens once, at the end of resolution.
func fillLevelSource(level *Level, source, model, effort string) {
	model, effort = strings.TrimSpace(model), strings.ToLower(strings.TrimSpace(effort))
	setPrimarySource(level, source, model, effort)
	fillModel(level, source, model)
	fillEffort(level, source, effort)
}

func setPrimarySource(level *Level, source, model, effort string) {
	if level.Source == "" && ((level.Model == "" && model != "") || (level.Effort == "" && effort != "")) {
		level.Source = source
	}
}

func fillModel(level *Level, source, model string) {
	if level.Model == "" && model != "" {
		level.Model, level.ModelSource = model, source
	}
}

func fillEffort(level *Level, source, effort string) {
	if level.Effort == "" && effort != "" {
		level.Effort, level.EffortSource = effort, source
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
	applyPolicyProvenance(&level, suggestion)
	// A model already answered by the host is shortened through the same
	// display table as a policy-selected one, so the label reads `Sonnet`
	// rather than `Claude-sonnet-5` whichever source supplied it.
	level.Model = policy.DisplayName(suggestion.Provider, level.Model)
	fillModel(&level, SourcePolicy, policy.DisplayName(suggestion.Provider, suggestion.Model))
	fillEffort(&level, SourcePolicy, suggestion.Effort)
	return level, nil
}

func applyPolicyProvenance(level *Level, suggestion Suggestion) {
	if level.Source == "" {
		level.Source = SourcePolicy
	}
	level.Provider, level.Capability = suggestion.Provider, suggestion.Capability
	level.FallbackUsed, level.FallbackReason = suggestion.FallbackUsed, suggestion.FallbackReason
	level.PolicyVersion, level.PolicyDigest = suggestion.PolicyVersion, suggestion.PolicyDigest
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
