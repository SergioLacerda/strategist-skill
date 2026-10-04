package application_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestResolveDigestInputUsesExactlyOneSource(t *testing.T) {
	t.Parallel()

	got, err := application.ResolveDigestInput("", "skill.md", func(string) (string, error) { return "sha256:file", nil })
	require.NoError(t, err)
	require.Equal(t, "sha256:file", got)
	_, err = application.ResolveDigestInput("sha256:flag", "skill.md", func(string) (string, error) { return "", nil })
	require.EqualError(t, err, "--resolved-digest and --file are mutually exclusive")
}

func TestCompareResolvedDigestDelegatesCatalogAuthority(t *testing.T) {
	t.Parallel()

	got, err := application.CompareResolvedDigest("brainstorming", "sha256:resolved", func(provider, resolved string) (application.ResolvedDigestComparison, error) {
		require.Equal(t, "brainstorming", provider)
		require.Equal(t, "sha256:resolved", resolved)
		return application.ResolvedDigestComparison{Status: application.ResolvedDigestMatch, Pin: resolved, Resolved: resolved}, nil
	})
	require.NoError(t, err)
	require.Equal(t, application.ResolvedDigestMatch, got.Status)

	_, err = application.CompareResolvedDigest("provider", "digest", func(string, string) (application.ResolvedDigestComparison, error) {
		return application.ResolvedDigestComparison{}, errors.New("catalog unavailable")
	})
	require.ErrorContains(t, err, "catalog unavailable")
}
