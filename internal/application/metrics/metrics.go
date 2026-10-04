// Package metricsapp owns the application port for explicit confidence gaps.
package metricsapp

import "fmt"

// MissingConfidenceInput is the application-level input for an explicit
// missing confidence record. Rendering and storage remain adapter concerns.
type MissingConfidenceInput struct {
	CorrelationKey string
	Reason         string
}

// ConfidenceMissingRecorder is implemented by the telemetry adapter.
type ConfidenceMissingRecorder interface {
	RecordMissing(correlationKey, reason string) error
}

// RecordMissingConfidence validates the explicit absence contract before the
// telemetry adapter persists it.
func RecordMissingConfidence(recorder ConfidenceMissingRecorder, input MissingConfidenceInput) error {
	if recorder == nil {
		return fmt.Errorf("metrics record: confidence recorder is required")
	}
	if input.CorrelationKey == "" || input.Reason == "" {
		return fmt.Errorf("metrics record: --missing requires --correlation-key and --reason")
	}
	if err := recorder.RecordMissing(input.CorrelationKey, input.Reason); err != nil {
		return fmt.Errorf("record missing confidence: %w", err)
	}
	return nil
}
