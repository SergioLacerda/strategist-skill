package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestParseDiscoveryProvenanceReadsStructureNotBody(t *testing.T) {
	got, err := ParseDiscoveryProvenance([]byte("---\nmission_id: m1\nmission_status: archivist_done\ninvocation_request_id: inv_a\n---\n\nmission_status: ranger_pending\n"))

	require.NoError(t, err)
	require.Equal(t, DiscoveryProvenance{MissionID: "m1", Status: "archivist_done", RequestID: "inv_a"}, got)
}

func TestParseDiscoveryProvenanceRejectsMissingOrMalformedFrontmatter(t *testing.T) {
	for _, raw := range []string{"body only", "---\nmission_status: [oops\n---\n", "---\nmission_status: x\n"} {
		_, err := ParseDiscoveryProvenance([]byte(raw))
		require.Error(t, err, raw)
	}
}

func embeddedNonceRequest(nonce string) DiscoveryWeaponRequest {
	request := validDiscoveryRequest()
	request.RuntimeKind = domain.RankedRuntimeEmbedded
	request.InvocationNonce = nonce
	return request
}

func embeddedNonceReceipt(request DiscoveryWeaponRequest, nonce string) connectors.EmbeddedInvocationReceipt {
	return connectors.EmbeddedInvocationReceipt{
		SchemaVersion: connectors.EmbeddedInvocationReceiptSchemaVersion, MissionID: request.MissionID, Role: request.Role,
		Slot: request.Slot, WeaponID: request.ProviderID, Entrypoint: "discover", BindingDigest: "sha256:b", SourceDigest: "sha256:s",
		Nonce: nonce, IssuedAt: time.Now(),
	}
}

func TestEmbeddedReceiptNonceMustMatchTheRequestNonce(t *testing.T) {
	request := embeddedNonceRequest("0123456789abcdef")

	require.NoError(t, validateEmbeddedInvocationReceipt(request, embeddedNonceReceipt(request, "0123456789abcdef")))
	require.ErrorContains(t, validateEmbeddedInvocationReceipt(request, embeddedNonceReceipt(request, "ffffffffffffffff")), "nonce mismatch")
	require.ErrorContains(t, validateEmbeddedInvocationReceipt(request, embeddedNonceReceipt(request, "")), "nonce mismatch")
}

func TestEmbeddedReceiptNonceIsOptionalWhenTheRequestHasNone(t *testing.T) {
	request := embeddedNonceRequest("")

	require.NoError(t, validateEmbeddedInvocationReceipt(request, embeddedNonceReceipt(request, "")))
}

func embeddedAdapterResponse(request DiscoveryWeaponRequest, adapter domain.MissionExecutionAdapter, policy, artifact string) DiscoveryWeaponResponse {
	receipt := embeddedNonceReceipt(request, request.InvocationNonce)
	receipt.RequestID, receipt.ExecutionAdapter, receipt.ChildPolicyID = "inv_abc", adapter, policy
	return DiscoveryWeaponResponse{ProviderID: request.ProviderID, InvocationEvidence: "embedded_prompt_bridge", Artifact: []byte(artifact), EmbeddedInvocationReceipt: receipt}
}

func TestEmbeddedDiscoveryStampsAdapterProvenanceAndDiscardsForgedClaims(t *testing.T) {
	request := embeddedNonceRequest("0123456789abcdef")
	request.ExecutionAdapter, request.ChildPolicyID = domain.ExecutionAdapterCodexChild, "strategist-child-policy/v1:codex:abc"
	forged := "---\nexecution_adapter: claude_child\nchild_policy_id: forged\ncapability_isolation: verified\n---\n\n# Findings\n"

	got, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return embeddedAdapterResponse(request, domain.ExecutionAdapterCodexChild, "strategist-child-policy/v1:codex:abc", forged), nil
	})

	require.NoError(t, err)
	content := string(got.Content)
	require.Contains(t, content, "execution_adapter: codex_child")
	require.Contains(t, content, "child_policy_id: strategist-child-policy/v1:codex:abc")
	require.Contains(t, content, "capability_isolation: unverified")
	require.NotContains(t, content, "forged")
	require.NotContains(t, content, "claude_child")
}

func TestEmbeddedDiscoveryWithoutAdapterCarriesNoProvenanceClaim(t *testing.T) {
	request := embeddedNonceRequest("")
	response := embeddedAdapterResponse(request, "", "", "---\ncapability_isolation: verified\n---\n\n# Findings\n")

	got, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return response, nil
	})

	require.NoError(t, err)
	require.NotContains(t, string(got.Content), "capability_isolation")
	require.NotContains(t, string(got.Content), "execution_adapter")
}

func TestEmbeddedDiscoveryRejectsAReceiptWhoseAdapterDiffersFromTheCommittedMode(t *testing.T) {
	request := embeddedNonceRequest("")
	request.ExecutionAdapter, request.ChildPolicyID = domain.ExecutionAdapterCurrentHost, ""
	for name, response := range map[string]DiscoveryWeaponResponse{
		"child upgrade":   embeddedAdapterResponse(request, domain.ExecutionAdapterCodexChild, "p", "# F\n"),
		"missing adapter": embeddedAdapterResponse(request, "", "", "# F\n"),
		"unknown adapter": embeddedAdapterResponse(request, "hacked", "", "# F\n"),
	} {
		_, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
			return response, nil
		})
		require.Error(t, err, name)
		require.Contains(t, err.Error(), "role_invocation_failed", name)
	}
}

func TestEmbeddedDiscoveryTelemetryReportsAdapterWithUnverifiedIsolationAndNoSecrets(t *testing.T) {
	request := embeddedNonceRequest("0123456789abcdef")
	request.ExecutionAdapter, request.ChildPolicyID = domain.ExecutionAdapterClaudeChild, "strategist-child-policy/v1:claude:abc"
	sink := &discoveryRecordingSink{}

	_, err := InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return embeddedAdapterResponse(request, domain.ExecutionAdapterClaudeChild, "strategist-child-policy/v1:claude:abc", "# Findings\n"), nil
	}, sink, "run-1")

	require.NoError(t, err)
	require.Len(t, sink.events, 1)
	attrs := sink.events[0].Attributes
	require.Equal(t, "claude_child", attrs[telemetry.AttrExecutionAdapter])
	require.Equal(t, "strategist-child-policy/v1:claude:abc", attrs[telemetry.AttrChildPolicyID])
	require.Equal(t, connectors.CapabilityIsolationUnverified, attrs[telemetry.AttrCapabilityIsolation])
	for key, value := range attrs {
		require.NotContains(t, fmt.Sprint(value), "0123456789abcdef", "nonce leaked through %s", key)
		require.NotContains(t, fmt.Sprint(value), "Findings", "output leaked through %s", key)
	}
}

func TestWithDiscoveryAdapterProvenanceOmitsAbsentValues(t *testing.T) {
	event := telemetry.WithDiscoveryAdapterProvenance(telemetry.NewDiscoveryWeaponEvent("r", "p", ".analysis/pending/m.md", "invoked", "normalized", "e", "", "embedded", "n/a", "unverified"), "current_host_adapter", "")

	require.Equal(t, "current_host_adapter", event.Attributes[telemetry.AttrExecutionAdapter])
	require.NotContains(t, event.Attributes, telemetry.AttrChildPolicyID)
}

func TestEmbeddedDiscoveryTelemetryCarriesTheRequestCorrelationAndRejectsAMismatchedReceipt(t *testing.T) {
	request := embeddedNonceRequest("")
	request.InvocationRequestID = "inv_abc"
	sink := &discoveryRecordingSink{}

	_, err := InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return embeddedAdapterResponse(request, "", "", "# F\n"), nil
	}, sink, "run")
	require.NoError(t, err)
	require.Equal(t, "inv_abc", sink.events[0].Attributes[telemetry.AttrInvocationRequestID])

	sink.events = nil
	other := request
	other.InvocationRequestID = "inv_other"
	_, err = InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), other, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return embeddedAdapterResponse(request, "", "", "# F\n"), nil
	}, sink, "run")
	require.ErrorContains(t, err, "request id mismatch")
	require.Equal(t, telemetry.DiscoveryInvocationFailed, sink.events[0].Attributes[telemetry.AttrDiscoveryInvocationStatus])
}

func TestWithDiscoveryRequestCorrelationOmitsAnEmptyIdentity(t *testing.T) {
	event := telemetry.NewDiscoveryWeaponEvent("r", "p", ".analysis/pending/m.md", "invoked", "normalized", "e", "", "embedded", "n/a", "unverified")

	require.NotContains(t, telemetry.WithDiscoveryRequestCorrelation(event, "").Attributes, telemetry.AttrInvocationRequestID)
	require.Equal(t, "inv_1", telemetry.WithDiscoveryRequestCorrelation(event, "inv_1").Attributes[telemetry.AttrInvocationRequestID])
}
