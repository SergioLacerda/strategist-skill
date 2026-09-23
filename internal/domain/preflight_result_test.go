package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLanguage(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want *PreflightLanguage
	}{
		{
			name: "structured map with all roles",
			raw: map[string]any{
				"ui": "pt-BR", "docs": "en", "chat": "pt-BR", "code": "en",
			},
			want: &PreflightLanguage{UI: "pt-BR", Docs: "en", Chat: "pt-BR", Code: "en"},
		},
		{name: "nil", raw: nil, want: nil},
		{name: "legacy scalar string", raw: "pt-BR", want: nil},
		{name: "empty map", raw: map[string]any{}, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ParseLanguage(tc.raw))
		})
	}
}
