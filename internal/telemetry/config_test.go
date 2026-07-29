package telemetry

import (
	"os"
	"testing"
)

func TestFromEnv_DefaultsInsecureToFalse(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")
	cfg := FromEnv()
	if cfg.Insecure {
		t.Error("Insecure must default to false when OTEL_EXPORTER_OTLP_INSECURE is unset")
	}
}

func TestFromEnv_DefaultServiceName(t *testing.T) {
	t.Parallel()
	os.Unsetenv("OTEL_SERVICE_NAME")
	cfg := FromEnv()
	if cfg.ServiceName != "strategist" {
		t.Errorf("expected default service name 'strategist', got %q", cfg.ServiceName)
	}
}

func TestFromEnv_defaults(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")

	cfg := FromEnv()
	if cfg.ServiceName != "strategist" {
		t.Errorf("expected default ServiceName=strategist, got %q", cfg.ServiceName)
	}
	if cfg.Insecure != false {
		t.Error("expected Insecure=false by default (TLS required)")
	}
	if cfg.Endpoint != "" {
		t.Errorf("expected empty Endpoint, got %q", cfg.Endpoint)
	}
}

func TestFromEnv_insecure_explicit_optin(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	cfg := FromEnv()
	if !cfg.Insecure {
		t.Error("expected Insecure=true when env is 'true'")
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
