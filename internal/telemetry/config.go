// Package telemetry provides OpenTelemetry setup and helpers for the strategist CLI.
package telemetry

import "os"

// Config holds telemetry configuration read from standard OTel environment variables.
type Config struct {
	Endpoint    string // OTEL_EXPORTER_OTLP_ENDPOINT
	ServiceName string // OTEL_SERVICE_NAME
	Insecure    bool   // OTEL_EXPORTER_OTLP_INSECURE (default true for dev)
}

// FromEnv reads OTel configuration from environment variables.
// If OTEL_SERVICE_NAME is unset, defaults to "strategist".
// If OTEL_EXPORTER_OTLP_INSECURE is unset or any non-"false" value, insecure is true.
func FromEnv() Config {
	svcName := os.Getenv("OTEL_SERVICE_NAME")
	if svcName == "" {
		svcName = "strategist"
	}
	insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") != "false"
	return Config{
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceName: svcName,
		Insecure:    insecure,
	}
}

// Enabled reports whether an OTel collector endpoint is configured.
// When false, Init installs a noop provider with zero overhead.
func (c Config) Enabled() bool {
	return c.Endpoint != ""
}
