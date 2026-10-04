package metricsapp_test

import (
	"errors"
	"testing"

	metricsapp "github.com/SergioLacerda/strategist-skill/internal/application/metrics"
	"github.com/stretchr/testify/require"
)

type missingRecorder struct {
	key, reason string
	err         error
}

func (r *missingRecorder) RecordMissing(key, reason string) error {
	r.key, r.reason = key, reason
	return r.err
}

func TestRecordMissingConfidenceValidatesAndDelegates(t *testing.T) {
	t.Parallel()

	recorder := &missingRecorder{}
	require.NoError(t, metricsapp.RecordMissingConfidence(recorder, metricsapp.MissingConfidenceInput{CorrelationKey: "boundary", Reason: "not produced"}))
	require.Equal(t, "boundary", recorder.key)
	require.Equal(t, "not produced", recorder.reason)
}

func TestRecordMissingConfidenceRejectsIncompleteInput(t *testing.T) {
	t.Parallel()

	err := metricsapp.RecordMissingConfidence(&missingRecorder{}, metricsapp.MissingConfidenceInput{Reason: "missing key"})
	require.ErrorContains(t, err, "correlation-key")

	err = metricsapp.RecordMissingConfidence(&missingRecorder{}, metricsapp.MissingConfidenceInput{CorrelationKey: "key"})
	require.ErrorContains(t, err, "reason")

	require.ErrorContains(t, metricsapp.RecordMissingConfidence(nil, metricsapp.MissingConfidenceInput{}), "recorder is required")
}

func TestRecordMissingConfidenceWrapsRecorderError(t *testing.T) {
	t.Parallel()

	err := metricsapp.RecordMissingConfidence(&missingRecorder{err: errors.New("telemetry unavailable")}, metricsapp.MissingConfidenceInput{
		CorrelationKey: "boundary",
		Reason:         "not produced",
	})
	require.EqualError(t, err, "record missing confidence: telemetry unavailable")
}
