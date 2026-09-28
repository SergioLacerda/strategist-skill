package weapon

import (
	"context"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRejectsUnknownAndIncompatibleAtomicWeapons(t *testing.T) {
	registry := Registry{"atomic": testWeapon("atomic")}
	cases := map[string]struct{ id, role, slot, want string }{
		"unknown":        {"nope", "ranger", "discovery", "was not found in catalog"},
		"wrong role":     {"atomic", "archivist", "discovery", "not compatible with Role"},
		"wrong slot":     {"atomic", "ranger", "refinement", "not compatible with slot"},
		"blank role":     {"atomic", " ", "discovery", "not compatible with Role"},
		"invalid Weapon": {"atomic", "ranger", "", "not compatible with slot"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := registry.Resolve(tc.id, tc.role, tc.slot)
			require.ErrorContains(t, err, tc.want)
		})
	}
	resolved, err := registry.Resolve("atomic", "ranger", "discovery")
	require.NoError(t, err)
	assert.Empty(t, resolved.Components)
}

func TestResolveRejectsInvalidManifest(t *testing.T) {
	broken := testWeapon("broken")
	broken.Origin = ""
	_, err := (Registry{"broken": broken}).Resolve("broken", "ranger", "discovery")
	require.ErrorContains(t, err, "origin")
}

func TestResolveComponentsGuardsDefensiveInvariants(t *testing.T) {
	registry := Registry{}
	parent := testWeapon("suite")
	parent.Kind = domain.WeaponKindComposite
	_, err := registry.resolveComponents(parent, "ranger", "discovery")
	require.ErrorContains(t, err, "has no composition")

	parent.Composition = &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{
		{ID: "a", DependsOn: []string{"b"}}, {ID: "b", DependsOn: []string{"a"}},
	}}
	_, err = registry.resolveComponents(parent, "ranger", "discovery")
	require.ErrorContains(t, err, "dependency cycle")
}

func TestResolveComponentRejectsUninvocableAndWrongSlotComponents(t *testing.T) {
	uninvocable := testWeapon("uninvocable")
	uninvocable.Runtime = domain.WeaponRuntime{Kind: domain.WeaponRuntimeNone}
	wrongSlot := testWeapon("wrong-slot")
	wrongSlot.SupportedSlots = []string{"refinement"}
	registry := Registry{"uninvocable": uninvocable, "wrong-slot": wrongSlot}

	_, err := registry.resolveComponent("suite", domain.WeaponComponent{ID: "uninvocable"}, "ranger", "discovery")
	require.ErrorContains(t, err, "not invocable")
	_, err = registry.resolveComponent("suite", domain.WeaponComponent{ID: "wrong-slot"}, "ranger", "discovery")
	require.ErrorContains(t, err, "incompatible with slot")
}

func atomicRequest() InvocationRequest {
	return InvocationRequest{MissionID: "mission-1", Role: "ranger", Slot: "discovery"}
}

func resolverFor(connector connectors.RuntimeConnector, err error) ConnectorResolver {
	return func(domain.WeaponManifest) (connectors.RuntimeConnector, error) { return connector, err }
}

func TestInvokeFailsClosedWhenConnectorCannotBeUsed(t *testing.T) {
	atomic := ResolvedWeapon{Manifest: testWeapon("atomic")}
	noInvoker := connectors.HostWeaponConnector{ConnectorID: "static", HostAPI: "strategist-host-skill/v1"}
	cases := map[string]struct {
		resolver ConnectorResolver
		want     string
	}{
		"nil resolver":   {nil, "resolver is unavailable"},
		"resolver error": {resolverFor(nil, errors.New("no connector")), "no connector"},
		"nil connector":  {resolverFor(nil, nil), "connector is nil"},
		"cannot invoke":  {resolverFor(noInvoker, nil), `connector "static" cannot invoke`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Invoke(context.Background(), atomic, atomicRequest(), tc.resolver)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestInvokeBlocksOnIdentityMismatchEvenForOptionalComponent(t *testing.T) {
	mismatch := fakeHost(func(connectors.InvocationEnvelope) connectors.ConnectorResult {
		return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: "impostor", InvocationEvidence: "evidence"}
	})
	optional := ResolvedWeapon{Manifest: testWeapon("suite"), Components: []ResolvedComponent{{Manifest: testWeapon("optional"), Required: false}}}
	optional.Manifest.Kind = domain.WeaponKindComposite

	outcome, err := Invoke(context.Background(), optional, atomicRequest(), resolverFor(mismatch, nil))
	require.ErrorContains(t, err, "does not match component")
	require.Len(t, outcome.Components, 1)
	assert.Equal(t, "incompatible_identity", outcome.Components[0].ReasonCode)
	assert.False(t, outcome.Degraded)
}

func TestInvokeDefaultsEntrypointAndReasonCode(t *testing.T) {
	var entrypoint string
	blocked := fakeHost(func(envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
		entrypoint = envelope.Entrypoint
		return connectors.ConnectorResult{Status: domain.ReadinessBlocked}
	})
	outcome, err := Invoke(context.Background(), ResolvedWeapon{Manifest: testWeapon("atomic")}, atomicRequest(), resolverFor(blocked, nil))
	require.Error(t, err)
	assert.Equal(t, "invoke", entrypoint)
	assert.Equal(t, "invocation_failed", outcome.Components[0].ReasonCode)
}

type recordingSink struct {
	events []telemetry.Event
	err    error
}

func (s *recordingSink) Emit(_ context.Context, event telemetry.Event) error {
	s.events = append(s.events, event)
	return s.err
}

func readyHost() connectors.RuntimeConnector {
	return fakeHost(func(envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
		return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: envelope.ComponentID, InvocationEvidence: " evidence "}
	})
}

func TestInvokeEmitsComponentTelemetryWithMissionFallbackRunID(t *testing.T) {
	sink := &recordingSink{}
	request := atomicRequest()
	request.Sink = sink
	outcome, err := Invoke(context.Background(), ResolvedWeapon{Manifest: testWeapon("atomic")}, request, resolverFor(readyHost(), nil))
	require.NoError(t, err)
	require.Len(t, sink.events, 1)
	assert.Equal(t, "evidence", outcome.Components[0].InvocationEvidence)

	request.RunID = "run-7"
	_, err = Invoke(context.Background(), ResolvedWeapon{Manifest: testWeapon("atomic")}, request, resolverFor(readyHost(), nil))
	require.NoError(t, err)
	require.Len(t, sink.events, 2)
}

func TestInvokeReportsTelemetryFailure(t *testing.T) {
	request := atomicRequest()
	request.Sink = &recordingSink{err: errors.New("sink down")}
	_, err := Invoke(context.Background(), ResolvedWeapon{Manifest: testWeapon("atomic")}, request, resolverFor(readyHost(), nil))
	require.ErrorContains(t, err, "emit Weapon component telemetry")
	require.ErrorContains(t, err, "sink down")
}
