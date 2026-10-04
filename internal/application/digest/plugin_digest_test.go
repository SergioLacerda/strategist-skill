package digest_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application/digest"
	"github.com/stretchr/testify/require"
)

func TestResolveDigestInputUsesExactlyOneSource(t *testing.T) {
	t.Parallel()

	got, err := digest.ResolveDigestInput("", "skill.md", func(string) (string, error) { return "sha256:file", nil })
	require.NoError(t, err)
	require.Equal(t, "sha256:file", got)
	_, err = digest.ResolveDigestInput("sha256:flag", "skill.md", func(string) (string, error) { return "", nil })
	require.EqualError(t, err, "--resolved-digest and --file are mutually exclusive")

	got, err = digest.ResolveDigestInput("sha256:flag", "", nil)
	require.NoError(t, err)
	require.Equal(t, "sha256:flag", got)

	_, err = digest.ResolveDigestInput("", "skill.md", nil)
	require.EqualError(t, err, "resolved-digest: file hash adapter is required")
	_, err = digest.ResolveDigestInput("", "skill.md", func(string) (string, error) {
		return "", errors.New("hash failed")
	})
	require.EqualError(t, err, "resolved-digest: hash failed")
}

func TestCompareResolvedDigestDelegatesCatalogAuthority(t *testing.T) {
	t.Parallel()

	got, err := digest.CompareResolvedDigest("brainstorming", "sha256:resolved", func(provider, resolved string) (digest.ResolvedDigestComparison, error) {
		require.Equal(t, "brainstorming", provider)
		require.Equal(t, "sha256:resolved", resolved)
		return digest.ResolvedDigestComparison{Status: digest.ResolvedDigestMatch, Pin: resolved, Resolved: resolved}, nil
	})
	require.NoError(t, err)
	require.Equal(t, digest.ResolvedDigestMatch, got.Status)

	_, err = digest.CompareResolvedDigest("provider", "digest", func(string, string) (digest.ResolvedDigestComparison, error) {
		return digest.ResolvedDigestComparison{}, errors.New("catalog unavailable")
	})
	require.ErrorContains(t, err, "catalog unavailable")

	_, err = digest.CompareResolvedDigest("provider", "digest", nil)
	require.EqualError(t, err, "resolved-digest: catalog comparison adapter is required")
}
