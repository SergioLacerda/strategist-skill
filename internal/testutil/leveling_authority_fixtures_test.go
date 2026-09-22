package testutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelingReasonCode(t *testing.T) {
	assert.Empty(t, LevelingReasonCode(nil))
	assert.Equal(t, "leveling_policy_missing", LevelingReasonCode(errors.New("error: leveling_policy_missing details")))
	assert.Empty(t, LevelingReasonCode(errors.New("unrelated error")))
}

func TestEmbeddedLevelingDefaults(t *testing.T) {
	defaults := EmbeddedLevelingDefaults(t)
	assert.NotEmpty(t, defaults)
	assert.Contains(t, string(defaults), "version:")
}

func TestLevelingAuthorityFixtures(t *testing.T) {
	fixtures := LevelingAuthorityFixtures()
	assert.NotEmpty(t, fixtures)

	for _, fix := range fixtures {
		t.Run(fix.Name, func(t *testing.T) {
			dir := t.TempDir()
			fix.Setup(t, dir)
			assert.NotEmpty(t, fix.Name)
		})
	}
}

func TestPortableStrategistRoot(t *testing.T) {
	root := PortableStrategistRoot(t)
	assert.DirExists(t, root)
	assert.Contains(t, root, ".strategist")
}
