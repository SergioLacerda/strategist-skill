package jeweler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func staticProviderEvidence() EvidenceVector {
	return EvidenceVector{
		Package: EvidenceCertified, Identity: EvidenceCertified,
		Provenance: EvidenceCertified, API: EvidenceCertified,
		Delegate: EvidenceCertified,
	}
}

func TestReadinessFromEvidenceFailsClosedForEachMissingDimension(t *testing.T) {
	checks := map[string]func(*EvidenceVector){
		"package":    func(e *EvidenceVector) { e.Package = EvidenceUnknown },
		"identity":   func(e *EvidenceVector) { e.Identity = EvidenceUnknown },
		"provenance": func(e *EvidenceVector) { e.Provenance = EvidenceUnknown },
		"api":        func(e *EvidenceVector) { e.API = EvidenceUnknown },
		"delegate":   func(e *EvidenceVector) { e.Delegate = EvidenceUnknown },
		"probe":      func(e *EvidenceVector) { e.Probe = EvidenceUnknown },
		"health":     func(e *EvidenceVector) { e.Health = EvidenceUnknown },
	}
	for name, mutate := range checks {
		evidence := staticProviderEvidence()
		evidence.Probe, evidence.Health = EvidenceCertified, EvidenceCertified
		mutate(&evidence)
		readiness := ReadinessFromEvidence(evidence)
		require.Equal(t, "unverified", readiness.Status, name)
		require.Equal(t, name+"_unverified", readiness.Reason, name)
		require.Equal(t, evidence, readiness.Evidence, name)
	}
}

func TestCLIProviderReadinessRequiresStaticAdmissionEvidence(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("provider status must not run without static admission evidence")
			return nil, nil
		},
	}

	readiness, err := provider.Readiness(context.Background(), "/workspace")
	require.NoError(t, err)
	require.Equal(t, "unverified", readiness.Status)
	require.Equal(t, "package_unverified", readiness.Reason)
}

func TestCLIProviderReadinessRequiresLiveProbeAndHealth(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Evidence:   staticProviderEvidence(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"Provider":"treasure-chest","Version":"0.2.0","SchemaVersion":1,"Healthy":true}`), nil
		},
	}

	readiness, err := provider.Readiness(context.Background(), "/workspace")
	require.NoError(t, err)
	require.Equal(t, "ready", readiness.Status)
	require.Equal(t, "activation_evidence_verified", readiness.Reason)
	require.Equal(t, EvidenceCertified, readiness.Evidence.Probe)
	require.Equal(t, EvidenceCertified, readiness.Evidence.Health)
}

func TestCLIProviderReadinessRejectsCallerSuppliedLiveEvidence(t *testing.T) {
	evidence := staticProviderEvidence()
	evidence.Probe = EvidenceStale
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Evidence:   evidence,
		Run: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("stale live evidence must not be refreshed through a caller-supplied claim")
			return nil, nil
		},
	}

	readiness, err := provider.Readiness(context.Background(), "/workspace")
	require.NoError(t, err)
	require.Equal(t, "blocked", readiness.Status)
	require.Equal(t, "live_evidence_not_host_collected", readiness.Reason)
}

func TestCLIProviderReadinessRejectsIdentityMismatch(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Evidence:   staticProviderEvidence(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"Provider":"other","Version":"0.2.0","SchemaVersion":1,"Healthy":true}`), nil
		},
	}

	readiness, err := provider.Readiness(context.Background(), "/workspace")
	require.NoError(t, err)
	require.Equal(t, "blocked", readiness.Status)
	require.Equal(t, "identity_blocked", readiness.Reason)
}

func TestCLIProviderReadinessReportsUnhealthyProvider(t *testing.T) {
	provider := CLIProvider{
		Executable: "treasure-chest",
		Identity:   validIdentity(),
		Evidence:   staticProviderEvidence(),
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"Provider":"treasure-chest","Version":"0.2.0","SchemaVersion":1,"Healthy":false}`), nil
		},
	}

	readiness, err := provider.Readiness(context.Background(), "/workspace")
	require.NoError(t, err)
	require.Equal(t, "degraded", readiness.Status)
	require.Equal(t, "health_failed", readiness.Reason)
}
