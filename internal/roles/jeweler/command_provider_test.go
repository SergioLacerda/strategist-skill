package jeweler

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIProviderQueryRequiresTheExternalEnvelope(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}

	_, err := provider.Query(context.Background(), validQuery())
	require.ErrorContains(t, err, "items are missing")
}

func TestCLIProviderQueryRejectsMismatchedEnvelope(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"Envelope":{"Provider":"other","Version":"0.2.0","SchemaVersion":1,"Freshness":"fresh","FallbackState":"none"},"Items":[]}`), nil
		},
	}

	_, err := provider.Query(context.Background(), validQuery())
	require.ErrorContains(t, err, "provider identity mismatch")
}

func TestCLIProviderQueryProjectsValidatedEnvelope(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"Envelope":{"Provider":"treasure-chest","Version":"0.2.0","SchemaVersion":1,"Freshness":"partially_stale","FallbackState":"none","Limitations":["partial"]},"Items":[]}`), nil
		},
	}

	projection, err := provider.Query(context.Background(), validQuery())
	require.NoError(t, err)
	require.Equal(t, FreshnessStale, projection.Freshness)
	require.Equal(t, []string{"partial"}, projection.Limitations)
}

func TestExternalFreshnessRejectsUnknownVocabulary(t *testing.T) {
	_, err := externalFreshness("eventually")
	require.ErrorContains(t, err, "freshness")
}

func TestCLIProviderMaintainPreservesPostOperationReadinessEvidence(t *testing.T) {
	var calls []string
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Evidence:   staticProviderEvidence(),
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			calls = append(calls, strings.Join(args, " "))
			if args[0] == "status" {
				return []byte(`{"Provider":"treasure-chest","Version":"0.2.0","SchemaVersion":1,"Healthy":true}`), nil
			}
			return []byte(`{"ok":true}`), nil
		},
	}

	result, err := provider.Maintain(context.Background(), MaintenanceRequest{
		Operation: "index", Root: "/workspace", Principal: "jeweler",
		CorrelationID: "c-1", Authorization: MaintenanceGrant,
	})
	require.NoError(t, err)
	require.Equal(t, "maintenance_command_completed", result.Readiness.Reason)
	require.Equal(t, EvidenceCertified, result.Readiness.Evidence.Probe)
	require.Equal(t, EvidenceCertified, result.Readiness.Evidence.Health)
	require.Equal(t, []string{
		"prepare --root /workspace --json",
		"status --root /workspace --json",
	}, calls)
}
