package otelsink_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	otelsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSink_Emit_ForwardsThroughDefaultSlog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	s := otelsink.New()
	err := s.Emit(context.Background(), telemetry.Event{
		Name:           "strategist.route.decision",
		Timestamp:      time.Now(),
		SeverityNumber: telemetry.SeverityInfo,
		Body:           "route selected",
		TraceID:        "trace-1",
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "route selected")
	assert.Contains(t, out, "trace_id=trace-1")
	assert.Contains(t, out, "severity_number=9")
}
