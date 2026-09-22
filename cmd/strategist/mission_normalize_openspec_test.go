package main

import (
	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePath(t *testing.T) {
	projectRoot := t.TempDir()
	fallback := filepath.Join(projectRoot, ".strategist", "openspec")

	assert.Equal(t, fallback, resolvePath("", fallback, projectRoot))
	absolute := filepath.Join(t.TempDir(), "custom")
	assert.Equal(t, absolute, resolvePath(absolute, fallback, projectRoot))
	assert.Equal(t, filepath.Join(projectRoot, "custom", "path"), resolvePath("custom/path", fallback, projectRoot))
}

func TestResolveNormalizePathsReportsMissingActiveConfig(t *testing.T) {
	_, _, _, err := resolveNormalizePaths(missionadapter.NormalizeOptions{Root: t.TempDir(), MissionID: "mission-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve active base path")
}
