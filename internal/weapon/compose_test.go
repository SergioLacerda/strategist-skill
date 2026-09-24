package weapon

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeHost(result func(connectors.InvocationEnvelope) connectors.ConnectorResult) connectors.RuntimeConnector {
	return connectors.HostWeaponConnector{ConnectorID: "fake-host", HostAPI: "strategist-host-skill/v1", Invoker: func(_ context.Context, envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
		return result(envelope)
	}}
}

func compositeFixture() ResolvedWeapon {
	parent := testWeapon("suite")
	parent.Kind = domain.WeaponKindComposite
	parent.Composition = &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{
		{ID: "required", Required: true}, {ID: "optional", Required: false, DependsOn: []string{"required"}},
	}}
	resolved, err := (Registry{parent.ID: parent, "required": testWeapon("required"), "optional": testWeapon("optional")}).Resolve("suite", "ranger", "discovery")
	if err != nil {
		panic(err)
	}
	return resolved
}

func TestInvokeCompositeWeaponPropagatesParentIdentityAndOrder(t *testing.T) {
	var ids []string
	outcome, err := Invoke(context.Background(), compositeFixture(), InvocationRequest{MissionID: "mission-1", Role: "ranger", Slot: "discovery", ArtifactPath: ".analysis/pending/mission-1.md"}, func(manifest domain.WeaponManifest) (connectors.RuntimeConnector, error) {
		return fakeHost(func(envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
			ids = append(ids, envelope.ComponentID)
			assert.Equal(t, "suite", envelope.WeaponID)
			assert.Equal(t, "mission-1/suite", envelope.ParentInvocationID)
			return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: manifest.ID, InvocationEvidence: "evidence-" + manifest.ID}
		}), nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"required", "optional"}, ids)
	assert.False(t, outcome.Degraded)
	assert.Equal(t, "mission-1/suite", outcome.ParentInvocationID)
}

func TestInvokeCompositeWeaponDegradesOnOptionalFailure(t *testing.T) {
	outcome, err := Invoke(context.Background(), compositeFixture(), InvocationRequest{MissionID: "mission-1", Role: "ranger", Slot: "discovery"}, func(manifest domain.WeaponManifest) (connectors.RuntimeConnector, error) {
		return fakeHost(func(connectors.InvocationEnvelope) connectors.ConnectorResult {
			if manifest.ID == "optional" {
				return connectors.ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "role_invocation_failed"}
			}
			return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: manifest.ID, InvocationEvidence: "evidence"}
		}), nil
	})
	require.NoError(t, err)
	assert.True(t, outcome.Degraded)
	assert.Equal(t, "role_invocation_failed", outcome.Components[1].ReasonCode)
}

func TestInvokeCompositeWeaponBlocksOnRequiredFailureAndMissingEvidence(t *testing.T) {
	_, err := Invoke(context.Background(), compositeFixture(), InvocationRequest{MissionID: "mission-1", Role: "ranger", Slot: "discovery"}, func(_ domain.WeaponManifest) (connectors.RuntimeConnector, error) {
		return fakeHost(func(connectors.InvocationEnvelope) connectors.ConnectorResult {
			return connectors.ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "required_failed"}
		}), nil
	})
	require.ErrorContains(t, err, `component "required" blocked`)

	_, err = Invoke(context.Background(), ResolvedWeapon{Manifest: testWeapon("atomic")}, InvocationRequest{MissionID: "mission-1", Role: "ranger", Slot: "discovery"}, func(domain.WeaponManifest) (connectors.RuntimeConnector, error) {
		return fakeHost(func(connectors.InvocationEnvelope) connectors.ConnectorResult {
			return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: "atomic"}
		}), nil
	})
	assert.ErrorContains(t, err, "invocation evidence is required")
}
