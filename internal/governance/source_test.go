package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSnapshot_NormalizesMandatesAndProvenance(t *testing.T) {
	source := fakeSource{name: "fixture"}
	snapshot, err := validateSnapshot(source, ".providence", Snapshot{
		SourceID:       "fixture",
		Fingerprint:    "fp",
		ActiveMandates: []string{"M002", "M001"},
		Validated:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"M001", "M002"}, snapshot.ActiveMandates)
	assert.Equal(t, ".providence", snapshot.Scope)
	assert.Equal(t, "fixture:fp", snapshot.CorrelationID)
}

func TestValidateSnapshot_RejectsIdentityMismatch(t *testing.T) {
	_, err := validateSnapshot(fakeSource{name: "fixture"}, ".providence", Snapshot{
		SourceID:    "other",
		Fingerprint: "fp",
		Validated:   true,
	})
	assert.ErrorContains(t, err, "identity mismatch")
}

func TestValidateSnapshot_RejectsMissingSourceIdentity(t *testing.T) {
	_, err := validateSnapshot(nil, ".providence", Snapshot{})
	require.ErrorContains(t, err, "source is required")

	_, err = validateSnapshot(fakeSource{}, ".providence", Snapshot{Fingerprint: "fp", Validated: true})
	assert.ErrorContains(t, err, "source name is required")
}

func TestValidateSnapshot_RejectsMissingScopeFingerprintAndMandate(t *testing.T) {
	source := fakeSource{name: "fixture"}
	_, err := validateSnapshot(source, ".providence", Snapshot{Validated: true})
	require.ErrorContains(t, err, "empty fingerprint")

	_, err = validateSnapshot(source, ".providence", Snapshot{Fingerprint: "fp"})
	require.ErrorContains(t, err, "unvalidated")

	_, err = validateSnapshot(source, "", Snapshot{Fingerprint: "fp", Validated: true})
	require.ErrorContains(t, err, "directory is required")

	_, err = validateSnapshot(source, ".providence", Snapshot{Fingerprint: "fp", Validated: true, ActiveMandates: []string{""}})
	assert.ErrorContains(t, err, "empty mandate")
}
