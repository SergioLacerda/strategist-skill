package check

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// buildPreflightResult aggregates this command's already-computed per-slot
// resolution and accumulated error/warning diagnostics into one
// domain.PreflightResult envelope (docs/adr/0041 D2). It performs no new
// checks of its own.
//
// Per-slot Bindings[].Status is derived from domain.PluginReadinessVector
// (task 2.5 — P4's RuntimeConnector SPI + readiness vector is confirmed
// wired into check_slots.go/check_readiness.go, so this is no longer the
// "interim" shortcut 2.1-2.3 shipped: a slot that resolved a provider but has
// a Blocked readiness dimension — e.g. entrypoint mismatch — is correctly
// reported blocked here too, not just "resolved therefore ready"). Warnings
// deliberately remains the full errs aggregate check.go's RunE assembled
// from every check category (readiness, rolevalidation, lock parity, persona,
// weapon bindings) — narrowing it to only readiness-vector messages would
// silently drop real, non-readiness-vector diagnostics (e.g. persona YAML
// errors) from --json output that the human-readable banner still reports.
// buildPreflightResult's status/exit-code semantics are governed by
// warnings only — advisories (preflightAdvisories: index_yaml_not_found,
// compiled_artifact_corrupt, directives_missing) are appended to the
// returned Warnings list for visibility but deliberately never influence
// status, matching those conditions' own "Non-blocking" behavior in
// contracts/machine/preflight.yaml.
func buildPreflightResult(root, mode string, providers map[string]string, resolutions map[string]slotResolution, warnings, advisories []string, language *domain.PreflightLanguage) domain.PreflightResult {
	status := "ready"
	if len(warnings) > 0 {
		status = "blocked"
	}

	bindings := make([]domain.PreflightBinding, 0, len(providers))
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		res, ok := resolutions[slot]
		bindingStatus := "ready"
		if !ok || len(blockedReadinessErrors(slot, res.readiness)) > 0 {
			bindingStatus = "blocked"
		}
		bindings = append(bindings, domain.PreflightBinding{
			Slot:     slot,
			Provider: providers[slot],
			Kind:     string(res.kind),
			Status:   bindingStatus,
		})
	}

	// Next carries a real phase token when ready (docs/adr/0044 DEC-001) —
	// intake is the only phase STARTUP ever proceeds to today, across every
	// route (00-routing.md's own sequences). The blocked branch keeps its
	// existing instructional message string, not a phase token.
	next := "intake"
	if status == "blocked" {
		next = "resolve the warnings below, then rerun `strategist check`"
	}

	allWarnings := make([]string, 0, len(warnings)+len(advisories))
	allWarnings = append(allWarnings, warnings...)
	allWarnings = append(allWarnings, advisories...)

	return domain.PreflightResult{
		SchemaVersion: domain.PreflightResultSchemaVersion,
		Status:        status,
		Identity:      domain.PreflightIdentity{Root: root, Mode: mode},
		Bindings:      bindings,
		Language:      language,
		Warnings:      allWarnings,
		Next:          next,
	}
}

// printPreflightJSON emits one PreflightResult JSON object to stdout for
// `strategist check --json`, and preserves check's existing exit-code
// semantics (non-zero when warnings is non-empty) — mirroring --simulate's
// behavior in check_simulate.go. It does not alter the default, human-readable
// output path.
func printPreflightJSON(root, mode string, providers map[string]string, resolutions map[string]slotResolution, warnings []string, language *domain.PreflightLanguage) error {
	result := buildPreflightResult(root, mode, providers, resolutions, warnings, append(preflightAdvisories(root), transitionalViewAdvisories(providers, resolutions)...), language)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("check --json: encode PreflightResult: %w", err)
	}
	if len(warnings) > 0 {
		return fmt.Errorf("[Strategist] check=failed errors=%d root=%s (json)", len(warnings), root)
	}
	return nil
}

// printPreflightJSONBlocked emits a minimal PreflightResult for a hard
// preflight failure detected before slot resolution begins — today, only
// check_identity.go's identity_files_missing block (its own, deliberately
// stricter D6 sibling of preflight.yaml's softer identity_files_missing
// condition). Without this, a --json caller hitting that early return would
// see a bare error string instead of a PreflightResult envelope; this keeps
// `strategist check --json` returning one consistent shape regardless of
// which check failed first. The returned error preserves the original
// blocking error's exit-code/message semantics.
func printPreflightJSONBlocked(root, mode string, blockingErr error, language *domain.PreflightLanguage) error {
	result := domain.PreflightResult{
		SchemaVersion: domain.PreflightResultSchemaVersion,
		Status:        "blocked",
		Identity:      domain.PreflightIdentity{Root: root, Mode: mode},
		Language:      language,
		Warnings:      []string{blockingErr.Error()},
		Next:          "resolve the warnings below, then rerun `strategist check`",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("check --json: encode PreflightResult: %w", err)
	}
	return blockingErr
}
