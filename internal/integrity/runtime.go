package integrity

import (
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/hardening"
)

// RuntimeResult is the unified outcome for compiled runtime integrity.
type RuntimeResult struct {
	hardening.Outcome
	Drift []string `json:"drift,omitempty"`
}

// VerifyRuntime composes the existing compiled-manifest verifier into one
// fail-closed, machine-readable result.
func VerifyRuntime(compiledDir string) RuntimeResult {
	drift, err := compile.VerifyManifest(compiledDir)
	if err != nil {
		return RuntimeResult{Outcome: hardening.Outcome{Status: hardening.StatusBlocked, Reason: "manifest_invalid", Remediation: "run strategist compile", Source: "compiled_manifest"}, Drift: []string{err.Error()}}
	}
	if len(drift) == 0 {
		return RuntimeResult{Outcome: hardening.Outcome{Status: hardening.StatusVerified, Reason: "manifest_verified", Source: "compiled_manifest"}}
	}
	status := hardening.StatusStale
	reason := "manifest_drift"
	if len(drift) == 1 && strings.Contains(drift[0], "not found") {
		status = hardening.StatusUnavailable
		reason = "manifest_unavailable"
	}
	return RuntimeResult{Outcome: hardening.Outcome{Status: status, Reason: reason, Remediation: "run strategist compile", Source: "compiled_manifest"}, Drift: drift}
}
