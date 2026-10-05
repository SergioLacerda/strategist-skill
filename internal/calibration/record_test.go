package calibration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func sealed(t *testing.T) Record {
	t.Helper()
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), DefaultConfig())
	require.Equal(t, StatusRecommended, report.Status)
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "jev-1.13.0", Profile: "handoff-v1", ProfileVersion: "abc12345", CorpusDigest: "sha256:c", CatalogVersion: CatalogVersion(), Seed: 1, Repetitions: 2, Now: fixedNow})
	require.NoError(t, err)
	return record
}

func TestNewRecordIsBoundAndSealed(t *testing.T) {
	record := sealed(t)
	assert.True(t, ValidID(record.ID))
	assert.Equal(t, "cal-20261004-", record.ID[:13])
	require.NotNil(t, record.Threshold)
	assert.NoError(t, record.Validate())
}

func TestARecordIsUsableOnlyForItsBinding(t *testing.T) {
	record := sealed(t)
	ok := Binding{Model: "jev-1.13.0", Profile: "handoff-v1", ProfileVersion: "abc12345"}
	assert.Equal(t, ReasonValid, record.UsableFor(ok, fixedNow.Add(24*time.Hour), DefaultMaxAge))

	other := ok
	other.Model = "jev-2.0.0"
	assert.Equal(t, ReasonModelChanged, record.UsableFor(other, fixedNow, DefaultMaxAge))
	other = ok
	other.ProfileVersion = "zzzz"
	assert.Equal(t, ReasonProfileChanged, record.UsableFor(other, fixedNow, DefaultMaxAge))
	assert.Equal(t, ReasonExpired, record.UsableFor(ok, fixedNow.Add(100*24*time.Hour), DefaultMaxAge))
}

func TestATamperedRecordIsInvalid(t *testing.T) {
	record := sealed(t)
	edited := *record.Threshold - 0.3
	record.Threshold = &edited
	assert.Equal(t, ReasonInvalid, record.UsableFor(Binding{Model: record.Model, Profile: record.Profile, ProfileVersion: record.ProfileVersion}, fixedNow, DefaultMaxAge))
	assert.Error(t, record.Validate())
}

func TestARecordWithoutARecommendationIsNeverUsed(t *testing.T) {
	report := Evaluate(runAll(t, corpus(8), FuncJudge(ruleJudge), 2), DefaultConfig())
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: "p", ProfileVersion: "v", Now: fixedNow})
	require.NoError(t, err)
	assert.Nil(t, record.Threshold)
	assert.Equal(t, ReasonNotRecommended, record.UsableFor(Binding{Model: "m", Profile: "p", ProfileVersion: "v"}, fixedNow, DefaultMaxAge))
}

func TestOnlyACalibrationIdShapeIsAnID(t *testing.T) {
	for _, bad := range []string{"latest", "", "cal-1", "../x", "cal-20261004-ZZZZZZZZ"} {
		assert.False(t, ValidID(bad), bad)
	}
	assert.True(t, ValidID("cal-20261004-0a1b2c3d"))
}

func TestNewRecordRequiresProvenance(t *testing.T) {
	_, err := NewRecord(Report{Status: StatusNotFit}, Meta{Now: fixedNow})
	assert.Error(t, err)
}
