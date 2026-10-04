package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/require"
)

func TestPreflightSubmitPreservesGuardOrderAndEvidence(t *testing.T) {
	var order []string
	ports := SubmitPreflightPorts{
		RequireNoAcceptedSideQuest:   func(string, string, domain.MissionEngineEvent) error { order = append(order, "side-quest"); return nil },
		RequireAuthoredPackageRepair: func(string, string, domain.MissionEngineEvent) error { order = append(order, "repair"); return nil },
		ValidateArtifacts:            func(string, string, domain.MissionEngineEvent) error { order = append(order, "artifacts"); return nil },
		ApprovalGatePackageDigest: func(string, domain.MissionEngineStatus, domain.MissionEngineEvent) (string, error) {
			order = append(order, "gate")
			return "sha256:gate", nil
		},
		RequireExecutionEvidence: func(string, string, domain.MissionEngineStatus, domain.MissionEngineEvent) (*handoff.Outcome, error) {
			order = append(order, "execution")
			return nil, nil
		},
		RepairPackageDigest: func(string, string, domain.MissionEngineEvent) (string, error) {
			order = append(order, "package")
			return "sha256:package", nil
		},
		PreflightSniperClaims: func(string, string, domain.MissionEngineEvent) ([]string, error) {
			order = append(order, "claims")
			return []string{"docs/adr/a.md"}, nil
		},
	}

	result, err := PreflightSubmit(SubmitPreflightRequest{Root: "root", BasePath: "base", MissionID: "m-1", Event: domain.MissionEventHandoffSatisfied}, ports)

	require.NoError(t, err)
	require.Equal(t, []string{"side-quest", "repair", "artifacts", "gate", "execution", "package", "claims"}, order)
	require.Equal(t, "sha256:gate", result.GateDigest)
	require.Equal(t, "sha256:package", result.PackageDigest)
	require.Equal(t, []string{"docs/adr/a.md"}, result.ClaimTargets)
}
