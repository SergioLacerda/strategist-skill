package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func productionDispatchRegistry() domain.CompiledRegistry {
	return domain.CompiledRegistry{
		SchemaVersion: domain.CompiledRegistrySchemaVersion,
		Weapons: []domain.CompiledWeapon{{
			ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon", SourceDigest: "sha256:source", Origin: domain.WeaponOriginEmbedded,
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, ConnectorID: "strategist-embedded", Entrypoint: "discover",
		}},
		Roles: []domain.CompiledRole{{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role"}},
		RankedBindings: []domain.CompiledRankedBinding{{
			Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon", SourceDigest: "sha256:source", RoleDigest: "sha256:role", BindingDigest: "sha256:binding", ExecutionMode: domain.WeaponExecutionModePromptBridge,
			CertificationDigest: "sha256:cert", ConnectorID: "strategist-embedded", Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, Entrypoint: "discover", Generation: 1, Status: "active",
		}},
	}
}

func TestNewProductionEmbeddedDispatchRegistersPromptBridge(t *testing.T) {
	dispatch, err := NewProductionEmbeddedDispatch(productionDispatchRegistry(), EmbeddedRuntimeDependencies{
		Payloads: func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("payload"), SourceDigest: "sha256:source"}, nil
		},
		PromptBridge: func(_ context.Context, request EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: request.Envelope.Instance.ID, InvocationEvidence: "embedded"}
		},
	})
	require.NoError(t, err)
	connector, err := dispatch.Connector("brainstorming", "1.0.0")
	require.NoError(t, err)
	result := connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming", Role: "ranger", Slot: "discovery", MissionID: "mission-1"})
	assert.Equal(t, domain.ReadinessReady, result.Status)
	assert.Equal(t, "embedded", result.InvocationEvidence)
}

func TestNewProductionEmbeddedDispatchRejectsMissingPromptBridge(t *testing.T) {
	_, err := NewProductionEmbeddedDispatch(productionDispatchRegistry(), EmbeddedRuntimeDependencies{})
	assert.ErrorContains(t, err, "prompt bridge for \"brainstorming\" is not registered")
}

func TestNewProductionEmbeddedDispatchBindsEachVersionToItsOwnPayloadAndBinding(t *testing.T) {
	registry := productionDispatchRegistry()
	newer := registry.Weapons[0]
	newer.Version, newer.Digest, newer.SourceDigest = "2.0.0", "sha256:weapon-2", "sha256:source-2"
	registry.Weapons = append(registry.Weapons, newer)
	binding := registry.RankedBindings[0]
	binding.WeaponVersion, binding.WeaponDigest, binding.SourceDigest, binding.BindingDigest, binding.CertificationDigest = "2.0.0", "sha256:weapon-2", "sha256:source-2", "sha256:binding-2", "sha256:cert-2"
	registry.RankedBindings = append(registry.RankedBindings, binding)
	require.NoError(t, registry.Validate())

	var requested []string
	var bound []string
	dispatch, err := NewProductionEmbeddedDispatch(registry, EmbeddedRuntimeDependencies{
		Payloads: func(_ context.Context, id, version string) (EmbeddedPromptPayload, error) {
			requested = append(requested, domain.WeaponIdentity(id, version))
			source := map[string]string{"1.0.0": "sha256:source", "2.0.0": "sha256:source-2"}[version]
			return EmbeddedPromptPayload{WeaponID: id, Content: []byte("payload " + version), SourceDigest: source}, nil
		},
		PromptBridge: func(_ context.Context, request EmbeddedPromptRequest) ConnectorResult {
			bound = append(bound, request.BindingDigest)
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: "brainstorming", InvocationEvidence: "embedded"}
		},
	})
	require.NoError(t, err)

	for _, version := range []string{"1.0.0", "2.0.0"} {
		connector, err := dispatch.Connector("brainstorming", version)
		require.NoError(t, err)
		result := connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming", WeaponVersion: version, Role: "ranger", Slot: "discovery", MissionID: "m"})
		assert.Equal(t, domain.ReadinessReady, result.Status, version)
	}
	assert.Equal(t, []string{"brainstorming@1.0.0", "brainstorming@2.0.0"}, requested, "each version reads its own payload")
	assert.Equal(t, []string{"sha256:binding", "sha256:binding-2"}, bound, "each version carries its own binding digest")
}

func TestEmbeddedPromptInvokerBlocksAnEnvelopeForAnotherVersion(t *testing.T) {
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:binding", "sha256:source",
		func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("p"), SourceDigest: "sha256:source"}, nil
		},
		func(context.Context, EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		})

	result := invoker(context.Background(), InvocationEnvelope{WeaponID: "brainstorming", WeaponVersion: "2.0.0"})

	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "embedded_prompt_weapon_mismatch", result.ReasonCode)
}
