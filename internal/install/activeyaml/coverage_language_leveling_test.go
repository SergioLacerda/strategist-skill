package activeyaml

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeStandaloneLanguageReplacesLegacyValues(t *testing.T) {
	got, err := NormalizeStandaloneLanguage([]byte("mode: epic\nlanguage: en\n"))
	require.NoError(t, err)
	assert.Contains(t, string(got), "ui: pt-BR")
	assert.Contains(t, string(got), "docs: en")
}

func TestReplaceStandaloneLanguageAppendsWhenMissing(t *testing.T) {
	got := replaceStandaloneLanguage([]byte("mode: epic\n"))
	assert.True(t, strings.HasSuffix(string(got), "language:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en\n"))
}

func TestLevelingActiveYAMLBranches(t *testing.T) {
	t.Run("unanswered", func(t *testing.T) {
		got, err := LevelingActiveYAML(domain.LevelingConfig{})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("manual", func(t *testing.T) {
		got, err := LevelingActiveYAML(domain.LevelingConfig{Mode: domain.LevelingModeManual})
		require.NoError(t, err)
		assert.Contains(t, got, "mode: manual")
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := LevelingActiveYAML(domain.LevelingConfig{Mode: "unknown"})
		assert.ErrorContains(t, err, "install:")
	})
}
