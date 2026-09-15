package conformance

import (
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RoleBindingTelemetryInput is the stable role/provider binding evidence
// payload — resolution, collision, migration, or conformance outcomes for
// the Role/Provider convergence model (see internal/domain/
// role_provider_contract.go and internal/plugins/role_binding.go).
// ProviderID and Source are empty for an outcome that never reached a
// resolved candidate (e.g. role_binding_missing, role_binding_ambiguous).
type RoleBindingTelemetryInput struct {
	// EventName names the outcome: "resolved", "id_shadowing",
	// "role_binding_missing", "role_binding_ambiguous", or
	// "migration_preview".
	EventName  string
	Role       string
	Slot       string
	ProviderID string
	Source     string
	Compatible bool
	ReasonCode string
}

// RoleBindingTelemetryEvent returns the canonical Strategist event envelope
// for one role/provider binding evidence point, following the same
// standalone-safe telemetry boundary as PluginTelemetryEvent — no external
// governance system or network call is required to construct or emit it
// (tasks.md Task 6.1: "Emit binding, resolution, collision, migration, and
// conformance evidence through existing standalone-safe telemetry/
// governance boundaries"; KF-08).
func RoleBindingTelemetryEvent(input RoleBindingTelemetryInput) telemetry.Event {
	return telemetry.Event{
		Name:           "strategist.role_binding." + input.EventName,
		Timestamp:      time.Now().UTC(),
		SeverityNumber: telemetry.SeverityInfo,
		Body:           input.ReasonCode,
		Attributes: map[string]any{
			"strategist.role_binding.role":        input.Role,
			"strategist.role_binding.slot":        input.Slot,
			"strategist.role_binding.provider_id": input.ProviderID,
			"strategist.role_binding.source":      input.Source,
			"strategist.role_binding.compatible":  input.Compatible,
			"strategist.role_binding.reason_code": input.ReasonCode,
		},
	}
}
