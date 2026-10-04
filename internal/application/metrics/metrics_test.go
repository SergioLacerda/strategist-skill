package metricsapp_test

import (
	"testing"

	metricsapp "github.com/SergioLacerda/strategist-skill/internal/application/metrics"
	"github.com/stretchr/testify/require"
)

type missingRecorder struct {
	key, reason string
}

func (r *missingRecorder) RecordMissing(key, reason string) error {
	r.key, r.reason = key, reason
	return nil
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
}
