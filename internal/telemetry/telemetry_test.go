package telemetry

import (
	"context"
	"testing"
)

func TestFromEnv_defaults(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")

	cfg := FromEnv()
	if cfg.ServiceName != "strategist" {
		t.Errorf("expected default ServiceName=strategist, got %q", cfg.ServiceName)
	}
	if cfg.Insecure != true {
		t.Error("expected Insecure=true by default")
	}
	if cfg.Endpoint != "" {
		t.Errorf("expected empty Endpoint, got %q", cfg.Endpoint)
	}
}

func TestFromEnv_override(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "my-service")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")

	cfg := FromEnv()
	if cfg.ServiceName != "my-service" {
		t.Errorf("expected ServiceName=my-service, got %q", cfg.ServiceName)
	}
	if cfg.Insecure != false {
		t.Error("expected Insecure=false when env is 'false'")
	}
	if cfg.Endpoint != "localhost:4317" {
		t.Errorf("expected Endpoint=localhost:4317, got %q", cfg.Endpoint)
	}
}

func TestConfig_Enabled_false(t *testing.T) {
	cfg := Config{Endpoint: ""}
	if cfg.Enabled() {
		t.Error("expected Enabled()=false when Endpoint is empty")
	}
}

func TestConfig_Enabled_true(t *testing.T) {
	cfg := Config{Endpoint: "localhost:4317"}
	if !cfg.Enabled() {
		t.Error("expected Enabled()=true when Endpoint is set")
	}
}

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

func TestTracer_returns_nonnnil(t *testing.T) {
	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestSchema_constants(t *testing.T) {
	constants := []string{
		AttrPhase, AttrStatus, AttrSkill, AttrArtifact,
		AttrReason, AttrCacheHit, AttrTarget, AttrMandates,
	}
	for _, c := range constants {
		if c == "" {
			t.Errorf("schema constant must not be empty string")
		}
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
