package sink_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/external"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/jsonl"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/noop"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/otel"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSink_AllConcreteSinksImplementEventSink is acceptance check 6.2: sinks
// are swappable without altering domain code — every concrete sink satisfies
// the same telemetry.EventSink interface the domain depends on.
func TestSink_AllConcreteSinksImplementEventSink(t *testing.T) {
	t.Parallel()
	var sinks = []telemetry.EventSink{
		noop.New(),
		slogsink.New(),
		jsonlsink.New(t.TempDir() + "/e.jsonl"),
		otelsink.New(),
		externalsink.New(nil),
	}
	for _, s := range sinks {
		require.NoError(t, s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()}))
	}
}

// TestSelect_NilBridgeNeverBreaksFlow is acceptance check 6.1: absence of
// external governance (bridge == nil, the common/default case) must not
// break the pipeline.
func TestSelect_NilBridgeNeverBreaksFlow(t *testing.T) {
	t.Parallel()
	s := sink.Select(telemetry.Config{}, nil)
	require.NotNil(t, s)
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.NoError(t, err)
}

func TestSelect_BridgePresentWrapsInExternal(t *testing.T) {
	t.Parallel()
	// Selection must not panic or error just because a bridge is configured —
	// concrete correctness of the external wrap is covered by external_test.go.
	bridge := recordingBridge{}
	s := sink.Select(telemetry.Config{}, bridge)
	require.NotNil(t, s)
	require.NoError(t, s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()}))
}

type recordingBridge struct{}

func (recordingBridge) Evaluate(context.Context, governancebridge.GovernanceRequest) (governancebridge.GovernanceDecision, error) {
	return governancebridge.GovernanceDecision{Allowed: true}, nil
}

type failingSink struct{}

func (failingSink) Emit(context.Context, telemetry.Event) error { return errors.New("exporter down") }

// TestResilient_NonStrict_SwallowsError is acceptance check 6.6 (default branch):
// exporter failure must not derail the mission.
func TestResilient_NonStrict_SwallowsError(t *testing.T) {
	t.Parallel()
	s := sink.Resilient(failingSink{}, false)
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.NoError(t, err)
}

// TestResilient_Strict_PropagatesError is acceptance check 6.6 (opt-in branch):
// strict mode makes exporter failure blocking.
func TestResilient_Strict_PropagatesError(t *testing.T) {
	t.Parallel()
	s := sink.Resilient(failingSink{}, true)
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.ErrorContains(t, err, "exporter down")
}

func TestResilient_SuccessPassesThroughEitherMode(t *testing.T) {
	t.Parallel()
	for _, strict := range []bool{true, false} {
		s := sink.Resilient(noop.New(), strict)
		assert.NoError(t, s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()}))
	}
}
