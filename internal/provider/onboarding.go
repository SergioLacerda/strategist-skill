package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Reason is a stable, machine-readable validation or onboarding explanation.
type Reason struct {
	Code   string `json:"code" yaml:"code"`
	Detail string `json:"detail" yaml:"detail"`
}

// Report is the provider validation result. LiveInvocation is intentionally
// separate from static readiness and remains unknown until an authorized host
// probe supplies runtime evidence.
type Report struct {
	ProviderID     string                       `json:"provider_id" yaml:"provider_id"`
	Version        string                       `json:"version" yaml:"version"`
	Source         string                       `json:"source" yaml:"source"`
	PackageDigest  string                       `json:"package_digest" yaml:"package_digest"`
	AdapterDigest  string                       `json:"adapter_digest" yaml:"adapter_digest"`
	SupportedRoles []string                     `json:"supported_roles" yaml:"supported_roles"`
	SupportedSlots []string                     `json:"supported_slots" yaml:"supported_slots"`
	RequestedSlot  string                       `json:"requested_slot,omitempty" yaml:"requested_slot,omitempty"`
	Readiness      domain.PluginReadinessVector `json:"readiness" yaml:"readiness"`
	LiveInvocation domain.ReadinessCheck        `json:"live_invocation" yaml:"live_invocation"`
	Reasons        []Reason                     `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	Validated      bool                         `json:"validated" yaml:"validated"`
}

// Validate performs the read-only provider contract check.
func Validate(input, requestedSlot string) (Report, error) {
	source, reasons := loadSource(input)
	if len(reasons) == 0 {
		reasons = validateSource(source, requestedSlot)
	}
	report := buildReport(source, input, requestedSlot, reasons)
	if len(reasons) > 0 {
		return report, fmt.Errorf("provider validation failed: %s", reasons[0].Code)
	}
	return report, nil
}

func buildReport(source Source, input, requestedSlot string, reasons []Reason) Report {
	packageDigest := source.Package.Digest
	adapterDigest := digestBytes(source.Files[adapterManifestName])
	readiness := staticReadiness(requestedSlot, reasons)
	return Report{
		ProviderID:     source.Package.ID,
		Version:        source.Package.Version,
		Source:         filepath.Clean(input),
		PackageDigest:  packageDigest,
		AdapterDigest:  adapterDigest,
		SupportedRoles: append([]string(nil), source.Adapter.SupportedRoles...),
		SupportedSlots: append([]string(nil), source.Adapter.SupportedSlots...),
		RequestedSlot:  requestedSlot,
		Readiness:      readiness,
		LiveInvocation: domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "live_probe_not_run", Detail: "static validation never certifies provider invocation"},
		Reasons:        reasons,
		Validated:      len(reasons) == 0,
	}
}

func staticReadiness(slot string, reasons []Reason) domain.PluginReadinessVector {
	failed := reasonCodes(reasons)
	status := domain.ReadinessReady
	code := "contract_verified"
	if len(reasons) > 0 {
		status = domain.ReadinessBlocked
		code = failed[0]
	}
	contract := domain.ReadinessCheck{Status: status, ReasonCode: code, Detail: "package and adapter are evaluated without runtime invocation"}
	return domain.PluginReadinessVector{
		Descriptor:          contract,
		Source:              domain.ReadinessCheck{Status: status, ReasonCode: firstOr(failed, "source_verified")},
		Conformance:         contract,
		Trust:               domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "trust_policy_not_evaluated"},
		Dependencies:        domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "dependencies_not_resolved"},
		HostAPI:             domain.ReadinessCheck{Status: status, ReasonCode: firstOr(failed, "host_api_compatible")},
		Connector:           domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_path_connector"},
		Entrypoint:          domain.ReadinessCheck{Status: status, ReasonCode: firstOr(failed, "entrypoint_declared")},
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "permission_grant_not_evaluated"},
		EnforcementCoverage: domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "enforcement_not_observed"},
		ActiveBinding:       bindingReadiness(slot, reasons),
	}
}

func bindingReadiness(slot string, reasons []Reason) domain.ReadinessCheck {
	if slot == "" {
		return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "slot_not_requested"}
	}
	if len(reasons) > 0 {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: reasonCodes(reasons)[0]}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "binding_not_activated"}
}

func reasonCodes(reasons []Reason) []string {
	codes := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if reason.Code != "" {
			codes = append(codes, reason.Code)
		}
	}
	return codes
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func ensureBindable(report Report, _ string) error {
	if !report.Validated {
		return fmt.Errorf("provider validation failed: %s", strings.Join(reasonCodes(report.Reasons), ", "))
	}
	// Static onboarding records a candidate binding only. Discovery invocation
	// remains a separate host boundary and is required at mission time; the
	// absence of live evidence must not turn a valid Weapon package into an
	// implicit native binding or prevent the operator from selecting it.
	return nil
}
