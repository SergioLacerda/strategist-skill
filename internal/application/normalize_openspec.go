package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// NormalizeOpenSpecRequest is the application-owned input for publishing a
// completed provider change. Path discovery and file loading stay with the
// composition root; the application service owns the use-case orchestration.
type NormalizeOpenSpecRequest struct {
	MissionID           string
	BasePath            string
	RuntimeRoot         string
	ChangeID            string
	PendingAnalysisPath string
	HandoffFacts        map[string]any
}

// NormalizeOpenSpecPublication is the stable publication event exposed to
// adapters. Refinement package types do not cross the application boundary.
type NormalizeOpenSpecPublication struct {
	MissionID        string
	ProviderChangeID string
	SourceDigest     string
	PackageDigest    string
	PublishedAt      time.Time
}

// NormalizeOpenSpecPorts connect publication telemetry to the composition
// root. Confidence is required; publication is optional for library callers.
type NormalizeOpenSpecPorts struct {
	RecordConfidence  func(domain.ConfidenceClaim, []domain.Evidence) error
	RecordPublication func(NormalizeOpenSpecPublication) error
}

// NormalizeOpenSpecResult is the consumer-facing result of publication.
type NormalizeOpenSpecResult struct {
	RefinedPath      string
	ProviderChangeID string
	ProviderRuntime  string
	PublishedAt      time.Time
}

// PublishOpenSpec publishes one completed provider change through the
// refinement boundary after validating the required application port.
func PublishOpenSpec(request NormalizeOpenSpecRequest, ports NormalizeOpenSpecPorts) (NormalizeOpenSpecResult, error) {
	if ports.RecordConfidence == nil {
		return NormalizeOpenSpecResult{}, errors.New("archivist confidence recorder is unavailable")
	}
	result, err := refinement.NormalizeOpenSpec(refinement.OpenSpecInput{
		MissionID: request.MissionID, BasePath: request.BasePath, RuntimeRoot: request.RuntimeRoot,
		ChangeID: request.ChangeID, PendingAnalysisPath: request.PendingAnalysisPath,
		HandoffFacts: request.HandoffFacts, RecordConfidence: ports.RecordConfidence,
		RecordPublication: func(publication refinement.PackagePublication) error {
			if ports.RecordPublication == nil {
				return nil
			}
			return ports.RecordPublication(NormalizeOpenSpecPublication{
				MissionID: publication.MissionID, ProviderChangeID: publication.ProviderChangeID,
				SourceDigest: publication.SourceDigest, PackageDigest: publication.PackageDigest,
				PublishedAt: publication.PublishedAt,
			})
		},
	})
	if err != nil {
		return NormalizeOpenSpecResult{}, fmt.Errorf("normalize OpenSpec: %w", err)
	}
	return NormalizeOpenSpecResult{
		RefinedPath: result.RefinedPath, ProviderChangeID: result.ProviderChangeID,
		ProviderRuntime: result.ProviderRuntime, PublishedAt: result.PublishedAt,
	}, nil
}

// AmendOpenSpecRequest is the application-owned input for a post-publication
// amendment. The persisted mission and gate state are loaded by the adapter
// and passed here as immutable authorization context.
type AmendOpenSpecRequest struct {
	MissionID           string
	BasePath            string
	RuntimeRoot         string
	ChangeID            string
	Amends              string
	AuthorizationRef    string
	GateLabel           string
	PersistedStatus     domain.MissionEngineStatus
	Reason              string
	SupersedesMissionID string
}

// AmendOpenSpecResult is the consumer-facing result of applying an amendment.
type AmendOpenSpecResult struct {
	RefinedPath      string
	AmendmentDir     string
	Amendment        int
	ProviderChangeID string
	Status           string
}

// ApplyOpenSpecAmendment applies one authorized amendment through refinement.
func ApplyOpenSpecAmendment(request AmendOpenSpecRequest) (AmendOpenSpecResult, error) {
	result, err := refinement.AmendOpenSpec(refinement.AmendInput{
		MissionID: request.MissionID, BasePath: request.BasePath, RuntimeRoot: request.RuntimeRoot,
		ChangeID: request.ChangeID, Amends: request.Amends, AuthorizationRef: request.AuthorizationRef,
		GateLabel: request.GateLabel, PersistedStatus: request.PersistedStatus, Reason: request.Reason,
		SupersedesMissionID: request.SupersedesMissionID,
	})
	if err != nil {
		return AmendOpenSpecResult{}, fmt.Errorf("amend OpenSpec: %w", err)
	}
	return AmendOpenSpecResult{
		RefinedPath: result.RefinedPath, AmendmentDir: result.AmendmentDir,
		Amendment: result.Amendment, ProviderChangeID: result.ProviderChangeID, Status: result.Status,
	}, nil
}
