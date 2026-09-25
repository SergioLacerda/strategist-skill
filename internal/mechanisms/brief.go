package mechanisms

import (
	"fmt"
	"strings"
)

// ForRole returns the rows a role should know about: those invoked by that role
// or by every role, in registry order.
func (r Registry) ForRole(role string) []Row {
	var rows []Row
	for _, row := range r.Rows {
		if invokedBy(row, role) {
			rows = append(rows, row)
		}
	}
	return rows
}

func invokedBy(row Row, role string) bool {
	for _, who := range row.InvokedBy {
		if who == role || who == AllRoles {
			return true
		}
	}
	return false
}

// Brief renders the role-scoped view: one compact line per row naming what it
// is and how to invoke it. The shipped-registry test keeps it small enough to
// pay for at every phase start (M005); BriefFull adds when to use each row and
// how it is enforced, for an agent that asks for the detail.
func (r Registry) Brief(role string) string { return r.render(role, false) }

// BriefFull is Brief with the enforcement and when-to-use detail of each row.
func (r Registry) BriefFull(role string) string { return r.render(role, true) }

func (r Registry) render(role string, full bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tools available to %s:\n", role)
	for _, row := range r.ForRole(role) {
		b.WriteString(briefLine(row, full))
	}
	return b.String()
}

func briefLine(row Row, full bool) string {
	line := fmt.Sprintf("- %s (%s): %s -> %s\n", row.ID, row.Family, row.Summary, row.HowToInvoke)
	if !full {
		return line
	}
	detail := "  [" + enforcement(row) + "]"
	if row.WhenToUse != "" {
		detail += " use: " + row.WhenToUse
	}
	return line + detail + "\n"
}

func enforcement(row Row) string {
	if row.EnforcementTier == "" {
		return row.EnforcementKind
	}
	return row.EnforcementKind + "/" + row.EnforcementTier
}
