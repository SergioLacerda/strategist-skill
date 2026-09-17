package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidNormalizeMissionID(t *testing.T) {
	for _, test := range []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid", id: "mission-123", want: true},
		{name: "empty", want: false},
		{name: "uppercase", id: "Mission-123", want: false},
		{name: "unsafe", id: "mission_123", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, validNormalizeMissionID(test.id))
		})
	}
}

func TestResolvePath(t *testing.T) {
	projectRoot := t.TempDir()
	fallback := filepath.Join(projectRoot, ".strategist", "openspec")

	assert.Equal(t, fallback, resolvePath("", fallback, projectRoot))
	absolute := filepath.Join(t.TempDir(), "custom")
	assert.Equal(t, absolute, resolvePath(absolute, fallback, projectRoot))
	assert.Equal(t, filepath.Join(projectRoot, "custom", "path"), resolvePath("custom/path", fallback, projectRoot))
}

func TestResolveNormalizePathsReportsMissingActiveConfig(t *testing.T) {
	_, _, _, err := resolveNormalizePaths(missionNormalizeOpenSpecOptions{Root: t.TempDir(), MissionID: "mission-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve active base path")
}
