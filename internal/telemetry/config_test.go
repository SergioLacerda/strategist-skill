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

func TestFromEnv_InsecureTrueWhenEnvSet(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	cfg := FromEnv()
	if !cfg.Insecure {
		t.Error("Insecure must be true when OTEL_EXPORTER_OTLP_INSECURE=true")
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
