package connectors

import (
	"context"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedPromptInvokerUsesOnlyTheEmbeddedPayload(t *testing.T) {
	bridgeCalled := false
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("# embedded skill"), SourceDigest: "sha256:source"}, nil
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:binding", "sha256:source", source, func(_ context.Context, request EmbeddedPromptRequest) ConnectorResult {
		bridgeCalled = true
		assert.Equal(t, []byte("# embedded skill"), request.Payload)
		assert.Equal(t, "sha256:binding", request.BindingDigest)
		return ConnectorResult{Status: domain.ReadinessReady, ProviderID: "brainstorming", InvocationEvidence: "embedded-prompt", Artifact: []byte("# findings")}
	})

	result := invoker(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming", Role: "ranger", Slot: "discovery", Entrypoint: "discover", MissionID: "mission-1",
	})

	require.Equal(t, domain.ReadinessReady, result.Status)
	assert.True(t, bridgeCalled)
	require.NoError(t, result.EmbeddedInvocationReceipt.Validate())
	assert.Equal(t, "sha256:source", result.EmbeddedInvocationReceipt.SourceDigest)
}

func TestEmbeddedPromptInvokerFailsClosedForBridgeOrPayloadDrift(t *testing.T) {
	tests := []struct {
		name   string
		source EmbeddedPromptPayloadSource
		bridge EmbeddedPromptBridge
		want   string
	}{
		{name: "missing bridge", source: func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{}, nil
		}, want: "embedded_prompt_bridge_unavailable"},
		{name: "missing source", bridge: func(context.Context, EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		}, want: "embedded_prompt_payload_unavailable"},
		{name: "wrong digest", source: func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("payload"), SourceDigest: "sha256:wrong"}, nil
		}, bridge: func(context.Context, EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		}, want: "embedded_prompt_payload_mismatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:binding", "sha256:source", test.source, test.bridge)
			result := invoker(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming", Role: "ranger", Slot: "discovery", MissionID: "mission-1"})
			assert.Equal(t, test.want, result.ReasonCode)
			assert.Equal(t, domain.ReadinessBlocked, result.Status)
		})
	}
}

func TestEmbeddedInvocationReceiptRejectsIncompleteIdentity(t *testing.T) {
	receipt := EmbeddedInvocationReceipt{SchemaVersion: EmbeddedInvocationReceiptSchemaVersion, IssuedAt: time.Now()}
	assert.ErrorContains(t, receipt.Validate(), "embedded invocation receipt mission is required")
}
