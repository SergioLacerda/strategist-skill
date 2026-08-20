package governancebridge_test

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBridge struct {
	decision governancebridge.GovernanceDecision
	err      error
}

func (f fakeBridge) Evaluate(context.Context, governancebridge.GovernanceRequest) (governancebridge.GovernanceDecision, error) {
	return f.decision, f.err
}

var _ governancebridge.GovernanceBridge = fakeBridge{}

func TestGovernanceBridge_Evaluate(t *testing.T) {
	t.Parallel()
	want := governancebridge.GovernanceDecision{Allowed: true, Authority: "external:sdd", CorrelationID: "c-1"}
	b := fakeBridge{decision: want}

	got, err := b.Evaluate(context.Background(), governancebridge.GovernanceRequest{
		MissionID: "m-1", Phase: "refinement", Action: "materialize", CorrelationID: "c-1",
	})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
