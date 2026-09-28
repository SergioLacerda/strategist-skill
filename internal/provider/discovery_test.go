package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type discoveryRecordingSink struct {
	events []telemetry.Event
	err    error
}

func (s *discoveryRecordingSink) Emit(_ context.Context, event telemetry.Event) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, event)
	return nil
}

func validDiscoveryRequest() DiscoveryWeaponRequest {
	return DiscoveryWeaponRequest{
		MissionID:    "mission-1",
		Role:         "ranger",
		Slot:         "discovery",
		ProviderID:   "brainstorming",
		ArtifactPath: ".analysis/pending/mission-1-analysis.md",
		ReceiptStore: NewMemoryReceiptNonceStore(),
	}
}

func TestInvokeAndNormalizeDiscoveryRejectsNonRangerBoundary(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		role string
		slot string
		want string
	}{
		{name: "wrong role", role: "archivist", slot: "discovery", want: "role must be ranger"},
		{name: "wrong slot", role: "ranger", slot: "refinement", want: "slot must be discovery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := validDiscoveryRequest()
			request.Role, request.Slot = tc.role, tc.slot
			_, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
				t.Fatal("invalid Ranger boundary must not invoke the Weapon")
				return DiscoveryWeaponResponse{}, nil
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "role_invocation_failed")
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func validDiscoveryResponse() DiscoveryWeaponResponse {
	return DiscoveryWeaponResponse{
		ProviderID:         "brainstorming",
		InvocationEvidence: "host-run-42",
		Artifact:           []byte("---\nmission_id: forged\nmission_status: documentation_applied\n---\n\n# Findings\n\nUntrusted result."),
		InvocationReceipt:  connectors.InvocationReceipt{SchemaVersion: connectors.InvocationReceiptSchemaVersion, MissionID: "mission-1", Role: "ranger", ProviderID: "brainstorming", ResolvedLocation: "skills/brainstorming/SKILL.md", ResolvedDigest: "sha256:test", Nonce: "nonce-1", IssuedAt: time.Now(), CapabilityIsolation: connectors.CapabilityIsolationUnverified},
	}
}

func TestInvokeAndNormalizeDiscoveryOverwritesProviderIdentityMetadata(t *testing.T) {
	t.Parallel()

	request := validDiscoveryRequest()
	got, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(_ context.Context, received DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		assert.Equal(t, request, received)
		return validDiscoveryResponse(), nil
	})
	require.NoError(t, err)
	assert.Equal(t, request.MissionID, got.MissionID)
	assert.Equal(t, request.ProviderID, got.ProviderID)
	assert.Equal(t, "host-run-42", got.InvocationEvidence)
	content := string(got.Content)
	assert.Contains(t, content, "schema_version: strategist-ranger-discovery/v1")
	assert.Contains(t, content, "mission_id: mission-1")
	assert.Contains(t, content, "mission_status: ranger_pending")
	assert.Contains(t, content, "analysis_artifact_path: .analysis/pending/mission-1-analysis.md")
	assert.Contains(t, content, "provider_id: brainstorming")
	assert.Contains(t, content, "# Findings")
	assert.NotContains(t, content, "mission_status: documentation_applied")
}

func TestInvokeAndNormalizeDiscoveryTrimsInvocationEvidenceConsistently(t *testing.T) {
	t.Parallel()

	response := validDiscoveryResponse()
	response.InvocationEvidence = "  host-run-42  "
	got, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return response, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "host-run-42", got.InvocationEvidence)
	assert.Contains(t, string(got.Content), "invocation_evidence: host-run-42")
}

func TestInvokeAndNormalizeDiscoveryFailsClosedWithoutInvoker(t *testing.T) {
	t.Parallel()

	_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "invoker is unavailable")
}

func TestInvokeAndNormalizeDiscoveryFailsClosedOnProviderError(t *testing.T) {
	t.Parallel()

	_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return DiscoveryWeaponResponse{}, errors.New("loader unavailable")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed: loader unavailable")
}

func TestInvokeAndNormalizeDiscoveryRejectsIncompatibleResult(t *testing.T) {
	t.Parallel()

	response := validDiscoveryResponse()
	response.ProviderID = "openspec-propose"
	_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return response, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "identity mismatch")
}

func TestInvokeAndNormalizeDiscoveryRejectsMissingEvidenceAndEmptyOutput(t *testing.T) {
	t.Parallel()

	t.Run("missing evidence", func(t *testing.T) {
		response := validDiscoveryResponse()
		response.InvocationEvidence = ""
		_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
			return response, nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invocation evidence is required")
	})

	t.Run("empty output", func(t *testing.T) {
		response := validDiscoveryResponse()
		response.Artifact = nil
		_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
			return response, nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty discovery artifact")
	})
}

func TestInvokeAndNormalizeDiscoveryRejectsMalformedFrontmatter(t *testing.T) {
	t.Parallel()

	response := validDiscoveryResponse()
	response.Artifact = []byte("---\nmission_id: broken\n# body")
	_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return response, nil
	})
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "role_invocation_failed:"))
	assert.Contains(t, err.Error(), "frontmatter is unclosed")
}

func TestInvokeAndNormalizeDiscoveryRejectsArtifactPathsOutsidePendingScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "traversal", path: ".analysis/pending/../refined/mission.md"},
		{name: "wrong prefix", path: ".analysis/refined/mission.md"},
		{name: "wrong extension", path: ".analysis/pending/mission.yaml"},
		{name: "absolute path", path: "/tmp/mission.md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := validDiscoveryRequest()
			request.ArtifactPath = tc.path
			_, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
				return validDiscoveryResponse(), nil
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "role_invocation_failed")
			assert.Contains(t, err.Error(), "slot_write_scope_violation")
		})
	}
}

func TestInvokeAndNormalizeDiscoveryWithTelemetryRecordsSuccessAndFailure(t *testing.T) {
	t.Parallel()

	sink := &discoveryRecordingSink{}
	_, err := InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return validDiscoveryResponse(), nil
	}, sink, "run-1")
	require.NoError(t, err)
	require.Len(t, sink.events, 1)
	assert.Equal(t, telemetry.DiscoveryInvocationInvoked, sink.events[0].Attributes[telemetry.AttrDiscoveryInvocationStatus])
	assert.Equal(t, telemetry.DiscoveryNormalizationNormalized, sink.events[0].Attributes[telemetry.AttrDiscoveryNormalization])
	assert.Equal(t, "host-run-42", sink.events[0].Attributes[telemetry.AttrInvocationEvidence])
	assert.Equal(t, "authenticated", sink.events[0].Attributes[telemetry.AttrReceiptAuthentication])
	assert.Equal(t, "pin_unavailable", sink.events[0].Attributes[telemetry.AttrReceiptPinStatus])
	assert.Equal(t, connectors.CapabilityIsolationUnverified, sink.events[0].Attributes[telemetry.AttrCapabilityIsolation])

	sink.events = nil
	_, err = InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), validDiscoveryRequest(), nil, sink, "run-1")
	require.Error(t, err)
	require.Len(t, sink.events, 1)
	assert.Equal(t, telemetry.DiscoveryInvocationFailed, sink.events[0].Attributes[telemetry.AttrDiscoveryInvocationStatus])
	assert.Equal(t, telemetry.DiscoveryNormalizationNotAttempted, sink.events[0].Attributes[telemetry.AttrDiscoveryNormalization])
	assert.Equal(t, "blocked", sink.events[0].Attributes[telemetry.AttrStatus])
}

func TestInvokeAndNormalizeDiscoveryWithTelemetryFailsClosedWhenSinkFails(t *testing.T) {
	t.Parallel()

	sink := &discoveryRecordingSink{err: errors.New("telemetry unavailable")}
	_, err := InvokeAndNormalizeDiscoveryWithTelemetry(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return validDiscoveryResponse(), nil
	}, sink, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "emit discovery telemetry")
}

func TestInvokeAndNormalizeDiscoveryRejectsInvalidInvocationReceipts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*DiscoveryWeaponResponse)
		want   string
	}{
		{name: "missing", mutate: func(response *DiscoveryWeaponResponse) { response.InvocationReceipt = connectors.InvocationReceipt{} }, want: "receipt"},
		{name: "mission mismatch", mutate: func(response *DiscoveryWeaponResponse) { response.InvocationReceipt.MissionID = "other" }, want: "identity mismatch"},
		{name: "role mismatch", mutate: func(response *DiscoveryWeaponResponse) { response.InvocationReceipt.Role = "archivist" }, want: "identity mismatch"},
		{name: "stale", mutate: func(response *DiscoveryWeaponResponse) {
			response.InvocationReceipt.IssuedAt = time.Now().Add(-6 * time.Minute)
		}, want: "stale"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := validDiscoveryResponse()
			tc.mutate(&response)
			_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
				return response, nil
			})
			require.ErrorContains(t, err, "role_invocation_failed")
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestInvokeAndNormalizeDiscoveryRejectsReplayedReceipt(t *testing.T) {
	t.Parallel()

	request := validDiscoveryRequest()
	response := validDiscoveryResponse()
	invoke := func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) { return response, nil }
	_, err := InvokeAndNormalizeDiscovery(context.Background(), request, invoke)
	require.NoError(t, err)
	_, err = InvokeAndNormalizeDiscovery(context.Background(), request, invoke)
	require.ErrorContains(t, err, "role_invocation_failed")
	require.ErrorContains(t, err, "replayed")
}

func TestInvokeAndNormalizeDiscoveryRejectsDeclaredPinMismatch(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "catalog.yaml")
	require.NoError(t, os.WriteFile(catalogPath, []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: brainstorming\n    risk_score: write_analysis\n    upstream_content_digest: sha256:pinned\n"), 0o600))
	request := validDiscoveryRequest()
	request.CatalogPath = catalogPath
	_, err := InvokeAndNormalizeDiscovery(context.Background(), request, func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return validDiscoveryResponse(), nil
	})
	require.ErrorContains(t, err, "role_invocation_failed")
	require.ErrorContains(t, err, "does not match declared pin")
}

func TestInvokeAndNormalizeDiscoveryRejectsUnprovenCapabilityIsolation(t *testing.T) {
	t.Parallel()

	response := validDiscoveryResponse()
	response.InvocationReceipt.CapabilityIsolation = connectors.CapabilityIsolationVerified
	_, err := InvokeAndNormalizeDiscovery(context.Background(), validDiscoveryRequest(), func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		return response, nil
	})
	require.ErrorContains(t, err, "role_invocation_failed")
	require.ErrorContains(t, err, "not host-verified")
}
