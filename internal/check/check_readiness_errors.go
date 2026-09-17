package check

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// readinessDimension names one PluginReadinessVector field for diagnostic
// messages, without requiring reflection or a change to the domain type.
type readinessDimension struct {
	name  string
	check domain.ReadinessCheck
}

func readinessDimensions(v domain.PluginReadinessVector) []readinessDimension {
	return []readinessDimension{
		{"descriptor", v.Descriptor},
		{"source", v.Source},
		{"trust", v.Trust},
		{"dependencies", v.Dependencies},
		{"host_api", v.HostAPI},
		{"connector", v.Connector},
		{"entrypoint", v.Entrypoint},
		{"permission_grant", v.PermissionGrant},
		{"enforcement_coverage", v.EnforcementCoverage},
		{"active_binding", v.ActiveBinding},
	}
}

// blockedReadinessErrors reports one message per dimension of slot's
// readiness vector that is explicitly domain.ReadinessBlocked — i.e. a
// dimension where the vector positively identifies an active problem, not
// merely one that is domain.ReadinessUnknown (not yet evaluated, by design —
// e.g. Trust/Dependencies/HostAPI/PermissionGrant are intentionally out of
// scope today) or domain.ReadinessUnsupported (the current runtime honestly
// does not offer that capability at all, e.g. an external skill plugin's
// Connector dimension). Gating strategist check's exit code on Blocked only
// — rather than on PluginReadinessVector.Ready(), which no configuration can
// satisfy today given those intentionally-unevaluated dimensions — means
// every currently-passing provider configuration keeps passing, while a
// configuration with a genuine, identifiable defect (e.g. an entrypoint
// manifest that doesn't exist or doesn't match its provider) newly fails
// with a precise slot+dimension+reason message instead of silently passing.
func blockedReadinessErrors(slot string, v domain.PluginReadinessVector) []string {
	var errs []string
	for _, d := range readinessDimensions(v) {
		if d.check.Status != domain.ReadinessBlocked {
			continue
		}
		msg := fmt.Sprintf("slot %s: readiness blocked on %s dimension (reason=%s", slot, d.name, d.check.ReasonCode)
		if d.check.Detail != "" {
			msg += ": " + d.check.Detail
		}
		msg += ")"
		errs = append(errs, msg)
	}
	return errs
}

// blockedReadinessErrorsForSlots applies blockedReadinessErrors across every
// slot in slots that has a resolution, collecting one error set. Extracted
// so check.go's RunE doesn't need to inline the per-slot loop itself.
func blockedReadinessErrorsForSlots(resolutions map[string]slotResolution, slots []string) []string {
	var errs []string
	for _, slot := range slots {
		res, ok := resolutions[slot]
		if !ok {
			continue
		}
		errs = append(errs, blockedReadinessErrors(slot, res.readiness)...)
	}
	return errs
}
