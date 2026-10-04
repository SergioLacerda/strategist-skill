package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// SubmitPreflightRequest identifies the event and durable package context that
// must be checked before the FSM can advance.
type SubmitPreflightRequest struct {
	Root      string
	BasePath  string
	MissionID string
	Status    domain.MissionEngineStatus
	Event     domain.MissionEngineEvent
}

// SubmitPreflightResult carries the evidence needed by the commit phase.
type SubmitPreflightResult struct {
	GateDigest    string
	Outcome       *handoff.Outcome
	PackageDigest string
	ClaimTargets  []string
}

// SubmitPreflightPorts keep filesystem, handoff, and artifact validators at
// the composition root while the application service owns their ordering.
type SubmitPreflightPorts struct {
	RequireNoAcceptedSideQuest   func(root, missionID string, event domain.MissionEngineEvent) error
	RequireAuthoredPackageRepair func(basePath, missionID string, event domain.MissionEngineEvent) error
	ValidateArtifacts            func(basePath, missionID string, event domain.MissionEngineEvent) error
	ApprovalGatePackageDigest    func(basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (string, error)
	RequireExecutionEvidence     func(root, basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (*handoff.Outcome, error)
	RepairPackageDigest          func(basePath, missionID string, event domain.MissionEngineEvent) (string, error)
	PreflightSniperClaims        func(basePath, missionID string, event domain.MissionEngineEvent) ([]string, error)
}

// PreflightSubmit validates the transition prerequisites in the same order as
// the legacy command path, returning only the evidence consumed by commit.
func PreflightSubmit(request SubmitPreflightRequest, ports SubmitPreflightPorts) (SubmitPreflightResult, error) {
	if err := validateSubmitPreflightPorts(ports); err != nil {
		return SubmitPreflightResult{}, err
	}
	if err := runSubmitPreflightGuards(request, ports); err != nil {
		return SubmitPreflightResult{}, err
	}
	return collectSubmitPreflightEvidence(request, ports)
}

func runSubmitPreflightGuards(request SubmitPreflightRequest, ports SubmitPreflightPorts) error {
	guards := []func() error{
		func() error { return ports.RequireNoAcceptedSideQuest(request.Root, request.MissionID, request.Event) },
		func() error {
			return ports.RequireAuthoredPackageRepair(request.BasePath, request.MissionID, request.Event)
		},
		func() error { return ports.ValidateArtifacts(request.BasePath, request.MissionID, request.Event) },
	}
	for _, guard := range guards {
		if err := guard(); err != nil {
			return err
		}
	}
	return nil
}

func collectSubmitPreflightEvidence(request SubmitPreflightRequest, ports SubmitPreflightPorts) (SubmitPreflightResult, error) {
	gateDigest, err := ports.ApprovalGatePackageDigest(request.BasePath, request.Status, request.Event)
	if err != nil {
		return SubmitPreflightResult{}, err
	}
	outcome, err := ports.RequireExecutionEvidence(request.Root, request.BasePath, request.Status, request.Event)
	if err != nil {
		return SubmitPreflightResult{}, err
	}
	packageDigest, err := ports.RepairPackageDigest(request.BasePath, request.MissionID, request.Event)
	if err != nil {
		return SubmitPreflightResult{}, err
	}
	claimTargets, err := ports.PreflightSniperClaims(request.BasePath, request.MissionID, request.Event)
	if err != nil {
		return SubmitPreflightResult{}, err
	}
	return SubmitPreflightResult{GateDigest: gateDigest, Outcome: outcome, PackageDigest: packageDigest, ClaimTargets: claimTargets}, nil
}

func validateSubmitPreflightPorts(ports SubmitPreflightPorts) error {
	if ports.RequireNoAcceptedSideQuest == nil || ports.RequireAuthoredPackageRepair == nil || ports.ValidateArtifacts == nil || ports.ApprovalGatePackageDigest == nil || ports.RequireExecutionEvidence == nil || ports.RepairPackageDigest == nil || ports.PreflightSniperClaims == nil {
		return fmt.Errorf("mission submit: preflight ports are required")
	}
	return nil
}
