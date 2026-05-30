package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/SergioLacerda/strategist-skill"

// Tracer returns the package-level tracer using the global TracerProvider.
// Always non-nil; returns a noop tracer when Init has not been called or
// when no OTel endpoint is configured.
func Tracer() trace.Tracer {
	return otel.Tracer(instrumentationName)
}
