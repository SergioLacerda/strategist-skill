package mission

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func missionEmbeddedRegistry() domain.CompiledRegistry {
	const brainstormingSourceDigest = "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608"
	return domain.CompiledRegistry{
		SchemaVersion: domain.CompiledRegistrySchemaVersion,
		Weapons: []domain.CompiledWeapon{{
			ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon", SourceDigest: brainstormingSourceDigest, Origin: domain.WeaponOriginEmbedded,
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, ConnectorID: "strategist-embedded", Entrypoint: "discover",
		}},
		Roles: []domain.CompiledRole{{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role"}},
		RankedBindings: []domain.CompiledRankedBinding{{
			Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon", SourceDigest: brainstormingSourceDigest, RoleDigest: "sha256:role", BindingDigest: "sha256:binding", ExecutionMode: domain.WeaponExecutionModePromptBridge,
			CertificationDigest: "sha256:cert", ConnectorID: "strategist-embedded", Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, Entrypoint: "discover", Generation: 1, Status: "active",
		}},
	}
}

func TestNewEmbeddedWeaponDispatchUsesEmbeddedBrainstormingPayload(t *testing.T) {
	registry := missionEmbeddedRegistry()
	dispatch, err := NewEmbeddedWeaponDispatch(registry, func(_ context.Context, request connectors.EmbeddedPromptRequest) connectors.ConnectorResult {
		assert.NotEmpty(t, request.Payload)
		assert.Equal(t, "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608", request.SourceDigest)
		return connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: request.Envelope.Instance.ID, InvocationEvidence: "embedded-prompt", Artifact: []byte("# Findings\n\nResult")}
	}, nil)
	require.NoError(t, err)
	connector, err := dispatch.Connector("brainstorming", "1.0.0")
	require.NoError(t, err)
	result := connector.Invoke(context.Background(), connectors.InvocationEnvelope{Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming", Role: "ranger", Slot: "discovery", Entrypoint: "discover", MissionID: "mission-1", BindingDigest: "sha256:binding", SourceDigest: "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608"})
	assert.Equal(t, domain.ReadinessReady, result.Status)
	assert.NoError(t, result.EmbeddedInvocationReceipt.Validate())
}

func TestNewEmbeddedWeaponDispatchRequiresPromptBridge(t *testing.T) {
	_, err := NewEmbeddedWeaponDispatch(missionEmbeddedRegistry(), nil, nil)
	assert.ErrorContains(t, err, "prompt bridge for \"brainstorming\" is not registered")
}
