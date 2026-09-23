package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Records written before run correlation and evidence_classes existed (no
// `run`, no `event_id`, singular `evidence_class`) must still decode, count
// for their mission and appear in a mission-wide review, while a run-scoped
// review excludes them.
func TestLegacyConfidenceRecordsStillDecodeAndAggregate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := ConfidenceHistoryPath(root)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	legacy := `{"mission_id":"m1","claim_id":"A-1","agent":"ranger","correlation_key":"k","claim_kind":"assertion","confidence_level":"high","confidence_percent":90,"evidence_ids":["E-1"],"evidence_class":"explicit","evidence_provided":true,"coverage_status":"reported","timestamp":"2026-08-01T00:00:00Z"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o600))

	records, err := ReadConfidenceRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1, "a legacy line decodes")
	assert.Empty(t, records[0].Run)
	assert.Equal(t, "explicit", records[0].EvidenceClass)

	review, err := LoadConfidenceGateReview(root, "m1")
	require.NoError(t, err)
	assert.Equal(t, 1, review.Metrics.SampleSize, "the legacy claim counts mission-wide")

	scoped, err := LoadConfidenceGateReviewForRun(root, "m1", "second")
	require.NoError(t, err)
	assert.Zero(t, scoped.Metrics.SampleSize, "a run-scoped review never adopts unscoped legacy records")
}
