// Package telemetry provides OpenTelemetry setup and helpers for the strategist CLI.
package telemetry

import "os"

// Config holds telemetry configuration read from standard OTel environment variables.
type Config struct {
	Endpoint    string // OTEL_EXPORTER_OTLP_ENDPOINT
	ServiceName string // OTEL_SERVICE_NAME
	Insecure    bool   // OTEL_EXPORTER_OTLP_INSECURE — default false (TLS required).
	// Set to "true" to allow plaintext gRPC (dev/self-hosted only).
	// Strict controls EventSink failure handling (STRATEGIST_TELEMETRY_STRICT).
	// Default false: a sink Emit error is logged and swallowed — telemetry
	// failure never blocks a mission (fail-open), matching the OTel SDK's own
	// error-handling spec ("MUST NOT cause the application to fail at runtime
	// due to dynamic config settings"). Set true to make Emit errors
	// propagate — useful in staging/CI to catch misconfiguration, the same
	// posture the OTel spec recommends ("MUST allow end users to change the
	// library's default error handling behavior... to run with strict error
	// handling in a staging environment").
	Strict bool
}

// FromEnv reads OTel configuration from environment variables.
// If OTEL_SERVICE_NAME is unset, defaults to "strategist".
// If OTEL_EXPORTER_OTLP_INSECURE is unset, insecure is false (TLS required).
// Set OTEL_EXPORTER_OTLP_INSECURE=true to allow plaintext connections.
func FromEnv() Config {
	svcName := os.Getenv("OTEL_SERVICE_NAME")
	if svcName == "" {
		svcName = "strategist"
	}
	insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true"
	strict := os.Getenv("STRATEGIST_TELEMETRY_STRICT") == "true"
	return Config{
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceName: svcName,
		Insecure:    insecure,
		Strict:      strict,
	}
}

// Enabled reports whether an OTel collector endpoint is configured.
// When false, Init installs a noop provider with zero overhead.
func (c Config) Enabled() bool {
	return c.Endpoint != ""
}
