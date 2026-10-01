package main

import (
	"fmt"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// verifyExecutionAdapter checks the adapter the calling command owns against
// the mode committed for the request before publication. A pre-field record
// only ever completes as the current host; a child mode is never inferred, and
// a request committed to a child cannot be completed through another path.
func verifyExecutionAdapter(record domain.MissionInvocationRecord, input missionadapter.InvocationCompleteInput) error {
	committed := record.EffectiveAdapter()
	switch {
	case input.Adapter == "":
		return fmt.Errorf("invocation_adapter_missing: the completion path did not declare an execution adapter")
	case !input.Adapter.Committable():
		return fmt.Errorf("invocation_adapter_unknown: %q is not an execution adapter", input.Adapter)
	case !committed.Known():
		return fmt.Errorf("invocation_adapter_unknown: request %q carries unrecognized adapter %q", record.Request.RequestID, committed)
	case committed == domain.ExecutionAdapterCurrentHostUnverified && input.Adapter == domain.ExecutionAdapterCurrentHost:
		return nil
	case committed != input.Adapter:
		return fmt.Errorf("invocation_adapter_mismatch: request %q was committed as %s but completion arrived as %s", record.Request.RequestID, committed, input.Adapter)
	case committed.IsChild() && record.ChildPolicyID == "":
		return fmt.Errorf("invocation_adapter_unknown: child request %q has no policy identity", record.Request.RequestID)
	}
	return nil
}
