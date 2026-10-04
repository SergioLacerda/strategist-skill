package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPublishOpenSpecRequiresConfidenceRecorder(t *testing.T) {
	_, err := PublishOpenSpec(NormalizeOpenSpecRequest{MissionID: "m-1"}, NormalizeOpenSpecPorts{})

	require.EqualError(t, err, "archivist confidence recorder is unavailable")
}

func TestPublishOpenSpecRejectsIncompleteInputThroughRefinement(t *testing.T) {
	_, err := PublishOpenSpec(NormalizeOpenSpecRequest{MissionID: "m-1"}, NormalizeOpenSpecPorts{
		RecordConfidence: func(domain.ConfidenceClaim, []domain.Evidence) error { return nil },
	})

	require.ErrorContains(t, err, "change id is malformed")
}

func TestApplyOpenSpecAmendmentPreservesRefinementValidation(t *testing.T) {
	_, err := ApplyOpenSpecAmendment(AmendOpenSpecRequest{MissionID: "m-1"})

	require.ErrorContains(t, err, "openspec amend")
}
