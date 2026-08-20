package slogsink_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	slogsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSink_Emit_LogsThroughDefaultSlog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	s := slogsink.New()
	err := s.Emit(context.Background(), telemetry.Event{
		Name:           "strategist.governance.decision",
		Timestamp:      time.Now(),
		SeverityNumber: telemetry.SeverityWarn,
		Body:           "mandate missing",
		Attributes:     map[string]any{telemetry.AttrEventContractID: "compile-domain"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "mandate missing")
	assert.Contains(t, out, "event.name=strategist.governance.decision")
	assert.Contains(t, out, "level=WARN")
}
