package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ProviderReason is the application-facing explanation emitted by provider
// validation and onboarding. Concrete provider packages map into this DTO.
type ProviderReason struct {
	Code   string `json:"code" yaml:"code"`
	Detail string `json:"detail" yaml:"detail"`
}

// ProviderReport is the consumer-owned provider result contract. It keeps
// provider implementation types out of the application package.
type ProviderReport struct {
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
	Reasons        []ProviderReason             `json:"reasons,omitempty" yaml:"reasons,omitempty"`
	Validated      bool                         `json:"validated" yaml:"validated"`
}

// ProviderAddResult is the application-facing committed onboarding result.
type ProviderAddResult struct {
	Report            ProviderReport `json:"report" yaml:"report"`
	InstanceID        string         `json:"instance_id" yaml:"instance_id"`
	BindingGeneration int64          `json:"binding_generation" yaml:"binding_generation"`
	TransactionState  string         `json:"transaction_state" yaml:"transaction_state"`
}

// ProviderPorts connect application orchestration to a concrete provider
// adapter without importing provider, filesystem, or CLI packages here.
type ProviderPorts struct {
	Validate func(source, requestedSlot string) (ProviderReport, error)
	Add      func(strategistRoot, source, slot string) (ProviderAddResult, error)
}

// ValidateProvider executes the read-only provider validation use case.
func ValidateProvider(source, requestedSlot string, ports ProviderPorts) (ProviderReport, error) {
	if source == "" {
		return ProviderReport{}, fmt.Errorf("provider source is required")
	}
	if ports.Validate == nil {
		return ProviderReport{}, fmt.Errorf("provider validation adapter is unavailable")
	}
	return ports.Validate(source, requestedSlot)
}

// AddProvider executes the provider onboarding use case through the supplied
// adapter. Filesystem staging, binding, and compilation remain adapter-owned.
func AddProvider(strategistRoot, source, slot string, ports ProviderPorts) (ProviderAddResult, error) {
	if strategistRoot == "" {
		return ProviderAddResult{}, fmt.Errorf("strategist root is required")
	}
	if source == "" {
		return ProviderAddResult{}, fmt.Errorf("provider source is required")
	}
	if slot == "" {
		return ProviderAddResult{}, fmt.Errorf("provider slot is required")
	}
	if ports.Add == nil {
		return ProviderAddResult{}, fmt.Errorf("provider onboarding adapter is unavailable")
	}
	return ports.Add(strategistRoot, source, slot)
}
