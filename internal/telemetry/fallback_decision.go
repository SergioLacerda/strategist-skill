package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// FallbackDecisionHistoryRelPath is relative to the .strategist runtime root.
const FallbackDecisionHistoryRelPath = "memory/fallback-decisions.jsonl"

// FallbackDecision is the durable record of a provider-fallback degradation
// event (ADR-0028, contracts/machine/provider-fallback.yaml): a mission phase
// where the configured slot provider was not invocable and Strategist
// proceeded using a compatible native role instead, either automatically
// (provider_resolution_policy=native) or after explicit user confirmation
// (provider_resolution_policy=ask). It is the structured, auditable
// counterpart to the narrative log line specified by
// contracts/narrative/00-routing.md § Provider Resolution Policy
// ("[Strategist] phase=<phase> status=degraded reason=native_fallback
// configured_provider=<x> effective_provider=<y>"), mirroring how
// RouteDecision is the durable counterpart to Scout's route_decision log
// line. A FallbackDecision is only ever recorded when a fallback was
// actually applied — outcome=blocked/unavailable never produce one, since
// nothing degraded. Append/read I/O for this record lives in
// fallback_decision_io.go.
type FallbackDecision struct {
	MissionID          string `json:"mission_id"`
	Slot               string `json:"slot"`
	Phase              string `json:"phase"`
	Policy             string `json:"policy"`
	Outcome            string `json:"outcome"`
	ConfiguredProvider string `json:"configured_provider"`
	EffectiveProvider  string `json:"effective_provider"`
	Reason             string `json:"reason"`
	UserConfirmed      bool   `json:"user_confirmed"`
	Degraded           bool   `json:"degraded"`
	Timestamp          string `json:"timestamp"`
}

// allowedFallbackSlots mirrors fallback-decision.schema.yaml#slot.allowed_values.
var allowedFallbackSlots = map[string]bool{
	"discovery":  true,
	"refinement": true,
	"execution":  true,
}

// allowedFallbackPolicies mirrors fallback-decision.schema.yaml#policy.allowed_values.
var allowedFallbackPolicies = map[string]bool{
	string(domain.ResolutionPolicyBlock):  true,
	string(domain.ResolutionPolicyAsk):    true,
	string(domain.ResolutionPolicyNative): true,
}

// allowedFallbackOutcomes mirrors fallback-decision.schema.yaml#outcome.allowed_values.
// Only the two outcomes that represent an applied fallback are valid here —
// blocked/unavailable/always_native never produce a FallbackDecision record.
var allowedFallbackOutcomes = map[string]bool{
	string(domain.FallbackOutcomeAskRequired): true,
	string(domain.FallbackOutcomeAutoNative):  true,
}

// FallbackDecisionHistoryPath returns the default runtime memory path for
// provider-fallback decision history.
func FallbackDecisionHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(FallbackDecisionHistoryRelPath))
}

// ValidateFallbackDecisionLine parses a single JSON line, checks required
// fields and allowed values per fallback-decision.schema.yaml, then
// cross-checks the claimed outcome against domain.ValidateFallbackDecision —
// the closed-loop check that catches a record misrepresenting what
// ADR-0028's policy table actually prescribes, or an ask_required outcome
// recorded without user confirmation.
func ValidateFallbackDecisionLine(line string) error {
	var d FallbackDecision
	if err := json.Unmarshal([]byte(line), &d); err != nil {
		return fmt.Errorf("fallback decision line is not valid JSON: %w", err)
	}
	var errs []error
	errs = append(errs, requiredFallbackField("mission_id", d.MissionID)...)
	errs = append(errs, allowedFallbackValue("slot", d.Slot, allowedFallbackSlots)...)
	errs = append(errs, requiredFallbackField("phase", d.Phase)...)
	errs = append(errs, allowedFallbackValue("policy", d.Policy, allowedFallbackPolicies)...)
	errs = append(errs, allowedFallbackValue("outcome", d.Outcome, allowedFallbackOutcomes)...)
	errs = append(errs, requiredFallbackField("configured_provider", d.ConfiguredProvider)...)
	errs = append(errs, requiredFallbackField("effective_provider", d.EffectiveProvider)...)
	errs = append(errs, requiredFallbackField("reason", d.Reason)...)
	errs = append(errs, requiredFallbackField("timestamp", d.Timestamp)...)
	if d.ConfiguredProvider != "" && d.ConfiguredProvider == d.EffectiveProvider {
		errs = append(errs, fmt.Errorf("configured_provider and effective_provider must differ for a degradation event"))
	}
	if !d.Degraded {
		errs = append(errs, fmt.Errorf("degraded must be true — a fallback decision record only exists when a fallback was actually applied"))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if err := domain.ValidateFallbackDecision(domain.FallbackDecisionFacts{
		Slot:          d.Slot,
		Policy:        domain.ResolutionPolicy(d.Policy),
		Outcome:       domain.FallbackOutcome(d.Outcome),
		UserConfirmed: d.UserConfirmed,
	}); err != nil {
		return fmt.Errorf("fallback decision domain validation failed: %w", err)
	}
	return nil
}

func requiredFallbackField(name, value string) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	return nil
}

func allowedFallbackValue(name, value string, allowed map[string]bool) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	if !allowed[value] {
		return []error{fmt.Errorf("%s %q is not an allowed value", name, value)}
	}
	return nil
}
