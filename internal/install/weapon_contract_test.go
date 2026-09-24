package install

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
		CanonicalRole:  "jewelcrafter",
		Roles:          []string{"jewelcrafter"},
		Kind:           domain.WeaponKindAtomic,
		SupportedSlots: []string{"discovery"},
		RiskScore:      "write_analysis",
		Runtime:        domain.RankedRuntimeContract{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
	})
	require.ErrorContains(t, err, "not approved for activation")
}

func TestPluginCatalogRejectsActiveWeaponWithoutRuntime(t *testing.T) {
	_, err := parseCatalogBytes([]byte(`
schema_version: strategist-plugin-catalog/v2
providers:
  - id: inactive-runtime
    version: "1.0.0"
    kind: atomic
    origin: embedded
    status: active
    risk_score: write_analysis
    roles: [ranger]
    supported_slots: [discovery]
    compatibility_source: embedded
    runtime:
      kind: none
`))
	require.ErrorContains(t, err, "active Weapon requires an invocable runtime")
}

func TestPluginCatalogRejectsLegacyRuntimeKindsAndSchema(t *testing.T) {
	entry := func(runtime string) string {
		return "schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: legacy\n    version: \"1.0.0\"\n    kind: atomic\n    origin: embedded\n    status: active\n    risk_score: write_analysis\n    roles: [ranger]\n    supported_slots: [discovery]\n    compatibility_source: embedded\n    runtime:\n      kind: " + runtime + "\n      host_api: strategist-host-skill/v1\n"
	}
	for _, legacy := range []string{"embedded_skill", "host_skill"} {
		_, err := parseCatalogBytes([]byte(entry(legacy)))
		require.ErrorContains(t, err, legacy)
		require.ErrorContains(t, err, "regenerate or reinstall")
	}
	_, err := parseCatalogBytes([]byte(strings.Replace(entry("host"), "catalog/v2", "catalog/v1", 1)))
	require.ErrorContains(t, err, "not supported")
	_, err = parseCatalogBytes([]byte(strings.Replace(entry("host"), "    origin: embedded\n", "", 1)))
	require.ErrorContains(t, err, "origin")
	_, err = parseCatalogBytes([]byte(entry("host")))
	require.NoError(t, err)
}
