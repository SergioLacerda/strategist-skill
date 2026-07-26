package telemetry

import (
	"context"
	"testing"
)

func TestTracer_returns_nonnnil(t *testing.T) {
	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestTracer_span_noop(_ *testing.T) {
	// With noop provider, starting and ending a span must not panic.
	cfg := Config{Endpoint: ""}
	_, _ = Init(cfg)

	ctx := context.Background()
	ctx, span := Tracer().Start(ctx, "test.span")
	_ = ctx
	span.End()
}
