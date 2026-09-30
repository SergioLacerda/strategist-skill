package embed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractorReadEmbeddedWeaponPayload(t *testing.T) {
	payload, digest, err := (Extractor{}).ReadEmbeddedWeaponPayload("brainstorming", "1.0.0")
	require.NoError(t, err)
	assert.NotEmpty(t, payload)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, digest)
}

func TestExtractorReadEmbeddedWeaponPayloadRejectsUnsafeID(t *testing.T) {
	for _, id := range []string{"", ".", "..", "../brainstorming", "nested/brainstorming", `nested\\brainstorming`} {
		t.Run(id, func(t *testing.T) {
			_, _, err := (Extractor{}).ReadEmbeddedWeaponPayload(id, "1.0.0")
			require.Error(t, err)
		})
	}
}

func TestExtractorReadEmbeddedWeaponPayloadRejectsUnsafeOrMissingVersion(t *testing.T) {
	for _, version := range []string{"", ".", "..", "../1.0.0", "1.0.0/x", `1.0.0\x`} {
		t.Run(version, func(t *testing.T) {
			_, _, err := (Extractor{}).ReadEmbeddedWeaponPayload("brainstorming", version)
			require.Error(t, err)
		})
	}
	_, _, err := (Extractor{}).ReadEmbeddedWeaponPayload("brainstorming@1.0.0", "1.0.0")
	require.Error(t, err, "the id may not smuggle a version")
	_, _, err = (Extractor{}).ReadEmbeddedWeaponPayload("brainstorming", "9.9.9")
	require.Error(t, err, "an unknown version never resolves to another one")
}
