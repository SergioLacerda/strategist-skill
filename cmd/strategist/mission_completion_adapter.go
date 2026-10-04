package main

import (
	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// verifyExecutionAdapter checks the adapter the calling command owns against
// the mode committed for the request before publication. A pre-field record
// only ever completes as the current host; a child mode is never inferred, and
// a request committed to a child cannot be completed through another path.
func verifyExecutionAdapter(record domain.MissionInvocationRecord, input missionadapter.InvocationCompleteInput) error {
	return wrapMissionError(application.VerifyExecutionAdapter(record, input.Adapter))
}
