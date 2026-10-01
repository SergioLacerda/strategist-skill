package connectors

// coverage_gaps_test.go – white-box tests that exercise branches not reached by
// the existing test suite. All tests run in the internal (same-package) view so
// unexported helpers can be called directly.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// binding_connector.go
// ---------------------------------------------------------------------------

func TestResolveRoleWeaponConnector_UnsupportedModeReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ResolveRoleWeaponConnector(context.Background(), domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "w", Mode: "unknown-mode",
	}, EmbeddedWeaponDispatch{}, nil)
	require.ErrorContains(t, err, "unsupported binding mode")
}

func TestResolveRankedConnector_NonEmbeddedRuntimeReturnsError(t *testing.T) {
	t.Parallel()
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"w@1.0.0": func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	})
	require.NoError(t, err)

	_, err = resolveRankedConnector(context.Background(), domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "w", WeaponVersion: "1.0.0",
		RuntimeKind: "host",
	}, dispatch)
	require.ErrorContains(t, err, "ranked_runtime_connector_unavailable")
}

func TestResolveCustomConnector_ConnectorCannotInvokeReturnsError(t *testing.T) {
	t.Parallel()
	noInvoke := LocalPathConnector{ConnectorID: "local-path", ConnectorAPIVersion: "strategist-connector-api/1"}
	_, err := resolveCustomConnector(context.Background(), domain.RoleWeaponBinding{WeaponID: "third-party"}, noInvoke)
	require.ErrorContains(t, err, "custom_runtime_connector_unavailable")
	require.ErrorContains(t, err, "cannot invoke")
}

func TestResolveCustomConnector_ValidConnectorIsReturned(t *testing.T) {
	t.Parallel()
	connector := HostWeaponConnector{
		ConnectorID: "host", ConnectorAPIVersion: "strategist-connector-api/1",
		Invoker: func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	}
	got, err := resolveCustomConnector(context.Background(), domain.RoleWeaponBinding{WeaponID: "third-party"}, connector)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

// ---------------------------------------------------------------------------
// embedded_weapon_dispatch.go
// ---------------------------------------------------------------------------

func TestNewEmbeddedWeaponDispatch_BlankWeaponIDIsRejected(t *testing.T) {
	t.Parallel()
	_, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"  ": func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	})
	require.ErrorContains(t, err, "Weapon ID is required")
}

func TestEmbeddedWeaponDispatchProbe_HappyPath(t *testing.T) {
	t.Parallel()
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"brainstorming@1.0.0": func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	})
	require.NoError(t, err)
	result := dispatch.Probe(context.Background(), "brainstorming", "1.0.0", "discover")
	assert.NotEqual(t, "embedded_weapon_dispatch_unavailable", result.ReasonCode)
}

func TestProductionCodeInvoker_RegisteredCodeInvokerIsReturned(t *testing.T) {
	t.Parallel()
	invoker := func(context.Context, InvocationEnvelope) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady}
	}
	got, err := productionCodeInvoker("my-weapon", map[string]EmbeddedWeaponInvoker{"my-weapon": invoker})
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestProductionCodeInvoker_MissingCodeInvokerReturnsError(t *testing.T) {
	t.Parallel()
	_, err := productionCodeInvoker("my-weapon", map[string]EmbeddedWeaponInvoker{})
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
	require.ErrorContains(t, err, "my-weapon")
}

func TestProductionEmbeddedInvoker_UnsupportedExecutionModeReturnsError(t *testing.T) {
	t.Parallel()
	registry := domain.CompiledRegistry{}
	weapon := domain.CompiledWeapon{
		ID: "w", Version: "1.0.0",
		Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: "unknown-mode"},
	}
	_, err := productionEmbeddedInvoker(registry, weapon, EmbeddedRuntimeDependencies{})
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
	require.ErrorContains(t, err, "unsupported execution mode")
}

func TestProductionEmbeddedInvoker_CodeModeIsRouted(t *testing.T) {
	t.Parallel()
	registry := domain.CompiledRegistry{}
	invoker := func(context.Context, InvocationEnvelope) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady}
	}
	weapon := domain.CompiledWeapon{
		ID: "my-code-weapon", Version: "1.0.0",
		Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode},
	}
	got, err := productionEmbeddedInvoker(registry, weapon, EmbeddedRuntimeDependencies{
		CodeInvokers: map[string]EmbeddedWeaponInvoker{"my-code-weapon": invoker},
	})
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestRankedBindingDigest_ReturnsEmptyWhenNotFound(t *testing.T) {
	t.Parallel()
	registry := domain.CompiledRegistry{}
	weapon := domain.CompiledWeapon{ID: "missing", Version: "1.0.0"}
	got := rankedBindingDigest(registry, weapon)
	assert.Empty(t, got)
}

func TestNewProductionEmbeddedDispatch_SkipsNonEmbeddedWeapons(t *testing.T) {
	t.Parallel()
	embedded := domain.CompiledWeapon{
		ID: "brainstorming", Version: "1.0.0", SourceDigest: "sha256:source",
		Runtime:    domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge},
		Entrypoint: "discover",
	}
	hostOnly := domain.CompiledWeapon{
		ID:      "host-tool",
		Version: "1.0.0",
		Runtime: domain.WeaponRuntime{Kind: "host"},
	}
	registry := domain.CompiledRegistry{
		Weapons: []domain.CompiledWeapon{hostOnly, embedded},
		RankedBindings: []domain.CompiledRankedBinding{{
			WeaponID: "brainstorming", WeaponVersion: "1.0.0", BindingDigest: "sha256:binding",
		}},
	}
	dispatch, err := NewProductionEmbeddedDispatch(registry, EmbeddedRuntimeDependencies{
		Payloads: func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("p"), SourceDigest: "sha256:source"}, nil
		},
		PromptBridge: func(context.Context, EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: "b", InvocationEvidence: "e"}
		},
	})
	require.NoError(t, err)
	_, err = dispatch.Connector("host-tool", "1.0.0")
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
}

func TestProductionPromptInvoker_MissingBindingDigestReturnsError(t *testing.T) {
	t.Parallel()
	registry := domain.CompiledRegistry{}
	weapon := domain.CompiledWeapon{
		ID: "brainstorming", Version: "1.0.0",
		Runtime: domain.WeaponRuntime{ExecutionMode: domain.WeaponExecutionModePromptBridge},
	}
	deps := EmbeddedRuntimeDependencies{
		PromptBridge: func(context.Context, EmbeddedPromptRequest) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
		Payloads: func(context.Context, string, string) (EmbeddedPromptPayload, error) {
			return EmbeddedPromptPayload{}, nil
		},
	}
	_, err := productionPromptInvoker(registry, weapon, deps)
	require.ErrorContains(t, err, "Ranked binding")
	require.ErrorContains(t, err, "is not registered")
}

// ---------------------------------------------------------------------------
// embedded_prompt_bridge.go
// ---------------------------------------------------------------------------

func TestInvokeEmbeddedPrompt_SourceErrorIsBlocked(t *testing.T) {
	t.Parallel()
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{}, errors.New("payload fetch failed")
	}
	bridge := func(context.Context, EmbeddedPromptRequest) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady}
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:b", "sha256:s", source, bridge)
	result := invoker(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming",
		Role: "ranger", Slot: "discovery", MissionID: "m",
	})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "embedded_prompt_payload_unavailable", result.ReasonCode)
	assert.Contains(t, result.Detail, "payload fetch failed")
}

func TestInvokeEmbeddedPrompt_BridgeReturnsBadStatusPropagated(t *testing.T) {
	t.Parallel()
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("x"), SourceDigest: "sha256:s"}, nil
	}
	bridge := func(context.Context, EmbeddedPromptRequest) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "bridge_error"}
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:b", "sha256:s", source, bridge)
	result := invoker(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming",
		Role: "ranger", Slot: "discovery", MissionID: "m",
	})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "bridge_error", result.ReasonCode)
}

func TestInvokeEmbeddedPrompt_IncompleteReadyResultIsBlocked(t *testing.T) {
	t.Parallel()
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("x"), SourceDigest: "sha256:s"}, nil
	}
	bridge := func(context.Context, EmbeddedPromptRequest) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady} // missing ProviderID and InvocationEvidence
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:b", "sha256:s", source, bridge)
	result := invoker(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"}, WeaponID: "brainstorming",
		Role: "ranger", Slot: "discovery", MissionID: "m",
	})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "embedded_prompt_result_incomplete", result.ReasonCode)
}

func TestValidateEmbeddedPromptRequest_WeaponIDMismatch(t *testing.T) {
	t.Parallel()
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("x"), SourceDigest: "sha256:s"}, nil
	}
	bridge := func(context.Context, EmbeddedPromptRequest) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady, ProviderID: "p", InvocationEvidence: "e"}
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:b", "sha256:s", source, bridge)
	result := invoker(context.Background(), InvocationEnvelope{WeaponID: "other-weapon"})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "embedded_prompt_weapon_mismatch", result.ReasonCode)
}

func TestValidateEmbeddedPromptRequest_HostAPIForbidden(t *testing.T) {
	t.Parallel()
	source := func(context.Context, string, string) (EmbeddedPromptPayload, error) {
		return EmbeddedPromptPayload{WeaponID: "brainstorming", Content: []byte("x"), SourceDigest: "sha256:s"}, nil
	}
	bridge := func(context.Context, EmbeddedPromptRequest) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady, ProviderID: "p", InvocationEvidence: "e"}
	}
	invoker := NewEmbeddedPromptInvoker("brainstorming", "1.0.0", "discover", "sha256:b", "sha256:s", source, bridge)
	result := invoker(context.Background(), InvocationEnvelope{HostAPI: "some-host-api"})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "embedded_host_api_forbidden", result.ReasonCode)
}

// ---------------------------------------------------------------------------
// invocation_receipt.go
// ---------------------------------------------------------------------------

func TestValidateInvocationReceipt_UnsupportedSchema(t *testing.T) {
	t.Parallel()
	receipt := InvocationReceipt{SchemaVersion: "wrong-schema"}
	err := ValidateInvocationReceipt(receipt)
	require.ErrorContains(t, err, "unsupported invocation receipt schema")
}

func TestValidateInvocationReceiptIdentity_IncompleteIdentity(t *testing.T) {
	t.Parallel()
	receipt := InvocationReceipt{
		SchemaVersion:    InvocationReceiptSchemaVersion,
		Role:             "ranger",
		ProviderID:       "p",
		ResolvedLocation: "skills/brainstorming",
		// MissionID missing
	}
	err := ValidateInvocationReceipt(receipt)
	require.ErrorContains(t, err, "identity is incomplete")
}

func TestValidateInvocationReceiptCompleteness_IssuedAtRequired(t *testing.T) {
	t.Parallel()
	receipt := InvocationReceipt{
		SchemaVersion:       InvocationReceiptSchemaVersion,
		MissionID:           "m",
		Role:                "ranger",
		ProviderID:          "p",
		ResolvedLocation:    "skills/brainstorming",
		ResolvedDigest:      "sha256:abc",
		Nonce:               "nonce",
		CapabilityIsolation: CapabilityIsolationUnverified,
		// IssuedAt deliberately zero
	}
	err := ValidateInvocationReceipt(receipt)
	require.ErrorContains(t, err, "incomplete")
}

func TestValidateInvocationReceiptCompleteness_InvalidCapabilityIsolation(t *testing.T) {
	t.Parallel()
	receipt := InvocationReceipt{
		SchemaVersion:       InvocationReceiptSchemaVersion,
		MissionID:           "m",
		Role:                "ranger",
		ProviderID:          "p",
		ResolvedLocation:    "skills/brainstorming",
		ResolvedDigest:      "sha256:abc",
		Nonce:               "nonce",
		IssuedAt:            time.Now(),
		CapabilityIsolation: "invalid-isolation",
	}
	err := ValidateInvocationReceipt(receipt)
	require.ErrorContains(t, err, "capability isolation is invalid")
}

func TestValidateInvocationReceipt_HappyPath(t *testing.T) {
	t.Parallel()
	receipt := InvocationReceipt{
		SchemaVersion:       InvocationReceiptSchemaVersion,
		MissionID:           "m",
		Role:                "ranger",
		ProviderID:          "p",
		ResolvedLocation:    "skills/brainstorming",
		ResolvedDigest:      "sha256:abc",
		Nonce:               "nonce",
		IssuedAt:            time.Now(),
		CapabilityIsolation: CapabilityIsolationVerified,
	}
	require.NoError(t, ValidateInvocationReceipt(receipt))
}

// ---------------------------------------------------------------------------
// EmbeddedInvocationReceipt.Validate – schema and IssuedAt branches
// ---------------------------------------------------------------------------

func TestEmbeddedInvocationReceiptValidate_UnsupportedSchema(t *testing.T) {
	t.Parallel()
	receipt := EmbeddedInvocationReceipt{SchemaVersion: "wrong-schema"}
	err := receipt.Validate()
	require.ErrorContains(t, err, "unsupported embedded invocation receipt schema")
}

func TestEmbeddedInvocationReceiptValidate_IssuedAtRequired(t *testing.T) {
	t.Parallel()
	receipt := EmbeddedInvocationReceipt{
		SchemaVersion: EmbeddedInvocationReceiptSchemaVersion,
		MissionID:     "m", Role: "r", Slot: "s", WeaponID: "w",
		Entrypoint: "e", BindingDigest: "b", SourceDigest: "sd",
		// IssuedAt deliberately zero
	}
	err := receipt.Validate()
	require.ErrorContains(t, err, "issued time is required")
}
