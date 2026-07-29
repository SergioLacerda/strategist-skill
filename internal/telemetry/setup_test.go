package telemetry

import (
	"context"
	"log/slog"
	"testing"
)

func TestInit_noop(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	cfg := Config{Endpoint: ""}
	shutdown, err := Init(cfg)
	if err != nil {
		t.Fatalf("expected nil error for noop init, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown func")
	}
	// shutdown must be callable without panicking
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("expected noop shutdown to return nil, got %v", err)
	}
}

func TestInit_invalid_endpoint(t *testing.T) {
	// An invalid endpoint causes the exporter to fail during New().
	// We rely on the fact that otlptracegrpc.New connects lazily — the error
	// may surface only at export time. This test just ensures Init does not panic.
	cfg := Config{
		Endpoint:    "not-a-real-host:9999",
		ServiceName: "test",
		Insecure:    true,
	}
	// The gRPC exporter connects lazily, so New() succeeds. Verify no panic.
	shutdown, err := Init(cfg)
	if err != nil {
		// Some implementations do fail eagerly — acceptable.
		t.Logf("Init returned error (acceptable for invalid endpoint): %v", err)
		return
	}
	if shutdown != nil {
		_ = shutdown(context.Background())
	}
}

func TestInit_enabled_success(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	cfg := Config{
		Endpoint:    "127.0.0.1:4317",
		ServiceName: "strategist-test",
		Insecure:    true,
	}

	shutdown, err := Init(cfg)
	if err != nil {
		t.Fatalf("expected enabled init to succeed, got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown func")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Logf("shutdown returned non-nil error (acceptable for a closed endpoint): %v", err)
	}
}
