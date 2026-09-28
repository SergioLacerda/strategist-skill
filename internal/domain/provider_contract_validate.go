package domain

import "fmt"

// Validate returns an error if required ProviderContract fields are missing
// or hold an unknown enum value.
func (p ProviderContract) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "schema_version", p.SchemaVersion)
	requireNonEmpty(&errs, "id", p.ID)
	requireNonEmpty(&errs, "version", p.Version)
	requireNonEmpty(&errs, "provider_schema_version", p.ProviderSchemaVersion)
	errs = append(errs, p.roleErrors()...)
	requireNonEmpty(&errs, "risk_score", p.RiskScore)
	errs = append(errs, p.enumErrors()...)
	if len(p.SupportedRoleContractVersions) == 0 {
		errs = append(errs, "supported_role_contract_versions must have at least one entry")
	}
	return joinPluginValidation("provider contract", errs)
}

// roleErrors reports a missing role declaration and invalid role references.
func (p ProviderContract) roleErrors() []string {
	var errs []string
	if p.CanonicalRole == "" && len(p.Roles) == 0 {
		errs = append(errs, "roles or canonical_role is required")
	}
	for _, roleID := range append([]string{p.CanonicalRole}, p.Roles...) {
		if err := ValidateRoleReference(roleID); err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

// enumErrors reports unknown source and materialization values.
func (p ProviderContract) enumErrors() []string {
	var errs []string
	if p.Source == "" {
		errs = append(errs, "source is required")
	} else if !hasString(validProviderSources, string(p.Source)) {
		errs = append(errs, fmt.Sprintf("source %q is not a known provider source", p.Source))
	}
	if p.Materialization != "" && !hasString(validMaterializationStates, string(p.Materialization)) {
		errs = append(errs, fmt.Sprintf("materialization %q is not a known materialization state", p.Materialization))
	}
	return errs
}
