// Package setup is the single planner of the operator's integration decision.
// The install wizard, silent install and the integrations commands all feed it
// the same selection and receive the same plan, so no adapter decides anything
// on its own and a credential that merely exists never enables a provider.
package setup

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
)

// Documented defaults proposed to the operator, never applied silently.
const (
	ProviderName         = "jev"
	DefaultEndpoint      = "https://api.typesafe.ai/v1/systemone"
	DefaultCredentialRef = "dotenv:.env#TYPESAFE_API_KEY" //nolint:gosec // G101: a reference to a variable name, not a credential
	DefaultModel         = "jev-1.13.0"
	defaultMaxStateBytes = 32768
)

// Choice is the operator's selection.
type Choice string

// Selections.
const (
	ChoiceEnable  Choice = "enable"
	ChoiceDisable Choice = "disable"
	ChoiceKeep    Choice = "keep"
)

// Action is what the plan does.
type Action string

// Plan actions.
const (
	ActionNone     Action = "none"
	ActionPreserve Action = "preserve"
	ActionEnable   Action = "enable"
	ActionDisable  Action = "disable"
)

// Overrides are explicit operator values that win over defaults and existing ones.
type Overrides struct {
	Endpoint      string
	CredentialRef string
	Model         string
}

// Input is everything the planner may use. It has no adapter-specific field.
type Input struct {
	Existing           config.File
	Found              bool
	Choice             Choice
	Overrides          Overrides
	CredentialResolves bool
}

// Plan is the deterministic result.
type Plan struct {
	Action Action
	File   config.File
	// Write is true only when the plan changes the persisted decision.
	Write bool
	// Pending is true when the provider is enabled but its credential does not
	// resolve yet; the intent is saved and diagnosed as credential_missing.
	Pending bool
	// Endpoint and DataCategories are what the operator is shown before consenting.
	Endpoint       string
	DataCategories []string
}

// Decide plans the integration decision for one selection.
func Decide(in Input) (Plan, error) {
	switch in.Choice {
	case "", ChoiceKeep:
		return preserve(in), nil
	case ChoiceEnable:
		return enable(in)
	case ChoiceDisable:
		return disable(in)
	default:
		return Plan{}, fmt.Errorf("integration choice %q is not one of enable, disable or keep", in.Choice)
	}
}

func preserve(in Input) Plan {
	if !in.Found {
		return Plan{Action: ActionNone}
	}
	return Plan{Action: ActionPreserve, File: in.Existing}
}

func base(in Input) config.File {
	file := config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{}}
	for name, provider := range in.Existing.Providers {
		file.Providers[name] = provider
	}
	return file
}

func enable(in Input) (Plan, error) {
	file := base(in)
	provider := file.Providers[ProviderName]
	provider.Enabled = true
	provider.Endpoint = pick(in.Overrides.Endpoint, provider.Endpoint, DefaultEndpoint)
	provider.CredentialRef = pick(in.Overrides.CredentialRef, provider.CredentialRef, DefaultCredentialRef)
	provider.Model = pick(in.Overrides.Model, provider.Model, DefaultModel)
	if len(provider.AllowedCapabilities) == 0 {
		provider.AllowedCapabilities = []integration.Capability{integration.CapHandoffValidate}
	}
	if len(provider.DataPolicy.AllowedFields) == 0 || provider.DataPolicy.MaxStateBytes <= 0 {
		provider.DataPolicy = config.DataPolicy{AllowedFields: DataCategories(), MaxStateBytes: defaultMaxStateBytes}
	}
	file.Providers[ProviderName] = provider
	if err := file.Validate(); err != nil {
		return Plan{}, fmt.Errorf("integration plan: %w", err)
	}
	return Plan{
		Action: ActionEnable, File: file, Write: true, Pending: !in.CredentialResolves,
		Endpoint: provider.Endpoint, DataCategories: provider.DataPolicy.AllowedFields,
	}, nil
}

func disable(in Input) (Plan, error) {
	file := base(in)
	provider := file.Providers[ProviderName]
	provider.Enabled = false
	file.Providers[ProviderName] = provider
	if err := file.Validate(); err != nil {
		return Plan{}, fmt.Errorf("integration plan: %w", err)
	}
	return Plan{Action: ActionDisable, File: file, Write: true}, nil
}

func pick(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
