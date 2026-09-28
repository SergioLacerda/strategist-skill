package domain

import "fmt"

// WeaponContract defines the role/provider boundary for a provider package.
// The contract is declarative evidence only: it does not certify that a host
// actually invoked the provider.
type WeaponContract struct {
	RoleOwner           string `yaml:"role_owner,omitempty"`
	Participation       string `yaml:"participation,omitempty"`
	InvocationEvidence  string `yaml:"invocation_evidence,omitempty"`
	UnavailableBehavior string `yaml:"unavailable_behavior,omitempty"`
	NativeSubstitution  string `yaml:"native_substitution,omitempty"`
}

// IsZero reports whether no weapon contract field is set.
func (w WeaponContract) IsZero() bool {
	return w.RoleOwner == "" && w.Participation == "" && w.InvocationEvidence == "" &&
		w.UnavailableBehavior == "" && w.NativeSubstitution == ""
}

// Validate accepts an unset contract or one that fully pins the required
// participation, evidence, failure and no-substitution values.
func (w WeaponContract) Validate() error {
	if w.IsZero() {
		return nil
	}
	if w.RoleOwner == "" {
		return fmt.Errorf("weapon contract role_owner is required")
	}
	if w.Participation != "required" {
		return fmt.Errorf("weapon contract participation must be \"required\", got %q", w.Participation)
	}
	if w.InvocationEvidence != "required" {
		return fmt.Errorf("weapon contract invocation_evidence must be \"required\", got %q", w.InvocationEvidence)
	}
	if w.UnavailableBehavior != "role_invocation_failed" {
		return fmt.Errorf("weapon contract unavailable_behavior must be \"role_invocation_failed\", got %q", w.UnavailableBehavior)
	}
	if w.NativeSubstitution != "forbidden" {
		return fmt.Errorf("weapon contract native_substitution must be \"forbidden\", got %q", w.NativeSubstitution)
	}
	return nil
}
