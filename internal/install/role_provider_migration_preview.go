package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RoleProviderPreviewEntry presents one slot's canonical Role separately
// from its compatible Provider candidates, each carrying its own
// provenance (Source) and materialization state independently of which one
// is currently selected (tasks.md Task 4.1; design.md Target architecture,
// EC-03: "no boolean native or installable may encode all lifecycle
// state").
type RoleProviderPreviewEntry struct {
	Slot              string
	RoleName          string
	CurrentProviderID string
	Candidates        []domain.ProviderContract
	Resolved          domain.ProviderBinding
	ResolutionError   string
}

// RoleProviderMigrationPreview previews migrating the legacy active.slots
// shape onto the Role/Provider/Binding model without mutating any
// workspace state (tasks.md Task 4.2). Nothing here writes active.yaml,
// SlotBinding, or PluginLock — it is a read-only projection over the
// catalog and roles/default.yaml.
type RoleProviderMigrationPreview struct {
	Entries []RoleProviderPreviewEntry
}

// FullyResolved reports whether every slot resolved to exactly one
// compatible Provider with no id_shadowing/ambiguity/missing error — the
// precondition tasks.md Task 4.2 requires before a migration may be
// applied ("apply only a fully resolved binding").
func (p RoleProviderMigrationPreview) FullyResolved() bool {
	if len(p.Entries) == 0 {
		return false
	}
	for _, entry := range p.Entries {
		if entry.ResolutionError != "" {
			return false
		}
	}
	return true
}

// Evidence renders one telemetry.Event per slot, recording the binding
// resolution outcome (resolved, id_shadowing, role_binding_missing, or
// role_binding_ambiguous) for every entry in the preview. It only builds
// the event envelopes — emitting them to a live EventSink is the caller's
// concern, the same division PluginTelemetryEvent already uses (tasks.md
// Task 6.1).
func (p RoleProviderMigrationPreview) Evidence() []telemetry.Event {
	events := make([]telemetry.Event, 0, len(p.Entries))
	for _, entry := range p.Entries {
		events = append(events, conformance.RoleBindingTelemetryEvent(conformance.RoleBindingTelemetryInput{
			EventName:  migrationEventName(entry),
			Role:       entry.RoleName,
			Slot:       entry.Slot,
			ProviderID: entry.Resolved.Provider.ID,
			Source:     string(entry.Resolved.Provider.Source),
			Compatible: entry.Resolved.Compatibility.Compatible,
			ReasonCode: entry.ResolutionError,
		}))
	}
	return events
}

func migrationEventName(entry RoleProviderPreviewEntry) string {
	if entry.ResolutionError == "" {
		return "resolved"
	}
	switch {
	case strings.HasPrefix(entry.ResolutionError, "unresolved_active_slot"):
		return "unresolved_active_slot"
	case strings.HasPrefix(entry.ResolutionError, "id_shadowing"):
		return "id_shadowing"
	case strings.HasPrefix(entry.ResolutionError, "role_binding_missing"):
		return "role_binding_missing"
	case strings.HasPrefix(entry.ResolutionError, "role_binding_ambiguous"):
		return "role_binding_ambiguous"
	default:
		return "role_binding_error"
	}
}

// Preview renders a human-readable migration preview, separating each
// slot's Role from its candidate Providers and marking provenance/current
// selection explicitly rather than collapsing them into one label.
func (p RoleProviderMigrationPreview) Preview() string {
	var b strings.Builder
	b.WriteString("role/provider migration preview\n")
	for _, entry := range p.Entries {
		fmt.Fprintf(&b, "slot=%s role=%s current_provider=%s\n", entry.Slot, entry.RoleName, entry.CurrentProviderID)
		for _, candidate := range entry.Candidates {
			marker := " "
			if candidate.ID == entry.CurrentProviderID {
				marker = "*"
			}
			fmt.Fprintf(&b, "  %s %s source=%s materialization=%s\n", marker, candidate.ID, candidate.Source, candidate.Materialization)
		}
		if entry.ResolutionError != "" {
			fmt.Fprintf(&b, "  BLOCKED: %s\n", entry.ResolutionError)
			continue
		}
		fmt.Fprintf(&b, "  resolved -> %s (compatible=%v)\n", entry.Resolved.Provider.ID, entry.Resolved.Compatibility.Compatible)
	}
	return b.String()
}
