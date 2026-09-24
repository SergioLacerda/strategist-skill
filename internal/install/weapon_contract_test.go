package install

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWeaponContractValidateRequiresFailClosedInvocation(t *testing.T) {
	contract := WeaponContract{
		RoleOwner:           "ranger",
		Participation:       "required",
		InvocationEvidence:  "required",
		UnavailableBehavior: "role_invocation_failed",
		NativeSubstitution:  "forbidden",
	}
	require.NoError(t, contract.Validate())

	contract.Participation = "advisory"
	require.ErrorContains(t, contract.Validate(), "participation")
	contract.Participation = "required"
	contract.UnavailableBehavior = "report_advisory"
	require.ErrorContains(t, contract.Validate(), "unavailable_behavior")
}

func TestWeaponContractValidateAllowsOmittedContract(t *testing.T) {
	require.NoError(t, (WeaponContract{}).Validate())
}

func TestExternalSkillAdapterRejectsUnverifiedRoleAffinity(t *testing.T) {
	err := validateExternalSkillAdapter("draft-role-weapon", externalSkillAdapter{
		CanonicalRole: "jewelcrafter",
		RiskScore:     "write_analysis",
	})
	require.ErrorContains(t, err, "not approved for activation")
}
