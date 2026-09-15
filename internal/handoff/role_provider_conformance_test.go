package handoff_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These fixtures back tasks.md Task 5 ("Protect role-owned handoffs and
// prove Provider conformance") for the Role/Provider convergence mission
// (.analysis/refined/strategist-papeis-personagens-skills-nativas/). They
// run entirely as deterministic Go assertions over existing structured
// types — no in-process LLM provider or ProviderResult API is introduced
// (Task 5.3; proposal.md Decision 8).

func sniperRole() domain.RoleContract {
	return domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          "sniper",
		Slot:          "execution",
		Must: []string{
			"materialize only the single declared active documentation target",
		},
		MustNot: []string{
			"materialize documentation without approval gate acceptance",
		},
	}
}

func embeddedSniperProvider() domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 "strategist-provider-contract/v1",
		ID:                            "sniper",
		Version:                       "1.0.0",
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 "sniper",
		RiskScore:                     "controlled",
		Source:                        domain.ProviderSourceNativeRole,
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
	}
}

// TestHandoffTypesCarryNoProviderIdentity proves proposal.md Decision 6
// ("Providers may emit intermediate data, but the Role produces canonical
// handoff artifacts") as a structural regression guard: none of the
// handoff package's public verification types may grow a field that lets
// a Provider's identity or provenance influence its own handoff
// verification outcome (Task 5.2: "without coupling Providers directly to
// downstream roles"). If this test starts failing, whoever added a
// Provider/Source field to one of these types has coupled a Role-owned
// boundary to a specific Provider and should reconsider the design instead
// of adjusting this fixture.
func TestHandoffTypesCarryNoProviderIdentity(t *testing.T) {
	t.Parallel()

	for _, v := range []any{handoff.Challenge{}, handoff.Acknowledgment{}, handoff.Policy{}, handoff.Result{}} {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			name := typ.Field(i).Name
			lower := strings.ToLower(name)
			assert.NotContains(t, lower, "provider",
				"%s.%s must not reference Provider identity — handoff verification is Role-owned (Decision 6)", typ.Name(), name)
		}
	}
}

// TestHandoffForbidsExecutionClaimRegardlessOfProviderSource proves that
// the Sniper role's MustNot — "materialize documentation without approval
// gate acceptance" — is enforced by handoff.Verify's ForbiddenClaims
// mechanism independently of which ProviderContract.Source resolved to the
// sniper role. This is the "forbidden implementation/approval behavior"
// fixture Task 5.1 asks for, run at the CLI/tool boundary (plain
// structured-data assertions, no in-process LLM call — Task 5.3).
func TestHandoffForbidsExecutionClaimRegardlessOfProviderSource(t *testing.T) {
	t.Parallel()

	role := sniperRole()
	require.Contains(t, role.MustNot, "materialize documentation without approval gate acceptance")

	binding := domain.ResolveProviderBinding(role, embeddedSniperProvider())
	require.True(t, binding.Compatibility.Compatible)

	for _, source := range []domain.ProviderSource{domain.ProviderSourceNativeRole, domain.ProviderSourceEmbedded, domain.ProviderSourceExternal} {
		provider := embeddedSniperProvider()
		provider.Source = source

		policy := handoff.DefaultPolicy()
		policy.ForbiddenClaims = []string{handoff.ForbiddenClaimExecutionAuthorized}

		gateFalselyClaimedAccepted := true
		ack := handoff.Acknowledgment{GateAllowed: &gateFalselyClaimedAccepted}

		result := handoff.Verify(policy, nil, ack)
		assert.Falsef(t, result.Passed, "source=%s: forbidden execution-authorized claim must fail verification regardless of Provider source", source)
		assert.Containsf(t, result.ForbiddenClaimViolations, handoff.ForbiddenClaimExecutionAuthorized, "source=%s", source)
	}
}

// TestHandoffRequiredChallengeTypesMatchRangerRoleContract proves the
// canonical handoff (proposal.md Decision 6) for the Ranger->Archivist
// transition still requires every challenge type the Ranger role contract
// implies, regardless of which candidate Provider (Brainstorm/embedded or
// an external migration candidate) would fill the ranger role — the
// required output shape is a Role property, not a Provider one (Task 5.1:
// "required outputs ... canonical handoffs"; Task 5.2: "Validate
// Brainstorm/Ranger ... as migration cases without coupling Providers
// directly to downstream roles").
func TestHandoffRequiredChallengeTypesMatchRangerRoleContract(t *testing.T) {
	t.Parallel()

	policy := handoff.RangerToArchivistPolicy()
	policy.Enabled = true

	challenges := []handoff.Challenge{
		{ID: "C-01", Type: handoff.ChallengeRecall, SourceRefs: []string{"KF-01"}},
		{ID: "C-02", Type: handoff.ChallengeBoundary, SourceRefs: []string{"KF-01"}},
		{ID: "C-03", Type: handoff.ChallengeClassification, ExpectedClassification: map[string]string{"KF-01": handoff.DecisionApproved}},
		{ID: "C-04", Type: handoff.ChallengeVerdict, SourceRefs: []string{"KF-01"}},
	}
	ack := handoff.Acknowledgment{
		UnderstoodRefs:  []string{"KF-01"},
		Classifications: map[string]string{"KF-01": handoff.DecisionApproved},
	}

	result := handoff.Verify(policy, challenges, ack)
	assert.True(t, result.Passed)

	// Dropping any one required challenge type must fail verification —
	// the required output set is exhaustive, not advisory, and this holds
	// no matter which Provider produced the underlying artifact.
	for i := range challenges {
		withoutOne := append([]handoff.Challenge(nil), challenges[:i]...)
		withoutOne = append(withoutOne, challenges[i+1:]...)
		result := handoff.Verify(policy, withoutOne, ack)
		assert.Falsef(t, result.Passed, "dropping challenge type %s must fail verification", challenges[i].Type)
		assert.Contains(t, result.MissingChallenges, challenges[i].Type)
	}
}
