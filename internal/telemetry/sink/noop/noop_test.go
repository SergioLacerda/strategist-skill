package noop_test

import (
	"context"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/noop"
	"github.com/stretchr/testify/require"
)

func TestSink_Emit_AlwaysNil(t *testing.T) {
	t.Parallel()
	s := noop.New()
	err := s.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})
	require.NoError(t, err)
}
