package i18n_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLang(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: i18n.LangEN},
		{input: "en", want: i18n.LangEN},
		{input: "EN", want: i18n.LangEN},
		{input: "pt-BR", want: i18n.LangPTBR},
		{input: "pt-br", want: i18n.LangPTBR},
		{input: "PT-BR", want: i18n.LangPTBR},
		{input: " pt ", want: i18n.LangPTBR},
		{input: "fr", want: "fr"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, i18n.NormalizeLang(tt.input), "input %q", tt.input)
	}
}

func TestBundleForFallsBackToEnglishForUnknownLanguage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, i18n.EN, i18n.BundleFor("fr"))
	assert.Equal(t, i18n.PT, i18n.BundleFor("PT-BR"))
}

func TestRuntimeBundleForUsesNormalizedLanguages(t *testing.T) {
	t.Parallel()

	ptBR, ok := i18n.RuntimeBundleFor("pt-br")
	require.True(t, ok)
	assert.Equal(t, i18n.PTBRRuntime, ptBR)

	en, ok := i18n.RuntimeBundleFor("en")
	require.True(t, ok)
	assert.Equal(t, i18n.ENRuntime, en)

	_, ok = i18n.RuntimeBundleFor("fr")
	assert.False(t, ok)
}

func TestPhaseAnnouncementsForUsesNormalizedLanguages(t *testing.T) {
	t.Parallel()

	ptBR, ok := i18n.PhaseAnnouncementsFor("PT-BR")
	require.True(t, ok)
	assert.Equal(t, i18n.PTBRPhaseAnnouncements, ptBR)

	_, ok = i18n.PhaseAnnouncementsFor("en")
	assert.False(t, ok)
}
