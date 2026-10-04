package leveling

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

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
