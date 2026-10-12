package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// EmbeddedRefinementRequest is the application-owned input for publishing one
// embedded Archivist completion. Path resolution and the invocation lease stay
// with the composition root.
type EmbeddedRefinementRequest struct {
	MissionID           string
	BasePath            string
	PendingAnalysisPath string
	ProviderID          string
	RequestID           string
	Result              string
}

// EmbeddedRefinementResult is the consumer-facing result of publication.
type EmbeddedRefinementResult struct {
	RefinedPath string
	PublishedAt time.Time
}

// PreparedEmbeddedRefinement is a completion that passed every validation and
// is ready to be written. Publication is a separate step so the caller can
// commit its invocation journal only for a completion that will publish.
type PreparedEmbeddedRefinement struct {
	prepared refinement.PreparedEmbedded
}

// PrepareEmbeddedRefinement parses the untrusted completion and validates it
// as the canonical package without writing anything. The Archivist confidence
// recorder is required, exactly as for the OpenSpec path (ADR-0067).
func PrepareEmbeddedRefinement(request EmbeddedRefinementRequest, ports NormalizeOpenSpecPorts) (PreparedEmbeddedRefinement, error) {
	if ports.RecordConfidence == nil {
		return PreparedEmbeddedRefinement{}, errors.New("archivist confidence recorder is unavailable")
	}
	completion, err := refinement.ParseEmbeddedCompletion(request.Result)
	if err != nil {
		return PreparedEmbeddedRefinement{}, fmt.Errorf("parse embedded refinement: %w", err)
	}
	prepared, err := refinement.PrepareEmbedded(refinement.EmbeddedInput{
		MissionID: request.MissionID, BasePath: request.BasePath, PendingAnalysisPath: request.PendingAnalysisPath,
		ProviderID: request.ProviderID, RequestID: request.RequestID, Completion: completion,
		RecordConfidence: ports.RecordConfidence,
		RecordAdvisories: func(input refinement.AdvisoryInput) error {
			if ports.RecordAdvisories == nil {
				return nil
			}
			return ports.RecordAdvisories(input)
		},
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
		return PreparedEmbeddedRefinement{}, fmt.Errorf("prepare embedded refinement: %w", err)
	}
	return PreparedEmbeddedRefinement{prepared: prepared}, nil
}

// Publish writes the prepared package. A completion already published by an
// earlier attempt of the same request is recognized and reported as success.
func (p PreparedEmbeddedRefinement) Publish() (EmbeddedRefinementResult, error) {
	result, err := p.prepared.Publish()
	if err != nil {
		return EmbeddedRefinementResult{}, fmt.Errorf("publish embedded refinement: %w", err)
	}
	return EmbeddedRefinementResult{RefinedPath: result.RefinedPath, PublishedAt: result.PublishedAt}, nil
}
