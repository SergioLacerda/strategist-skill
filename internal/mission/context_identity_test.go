package mission

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewContextIdentityIsCanonicalAndInvalidatesEveryInput(t *testing.T) {
	t.Parallel()
	base := ContextIdentityInput{
		SourceDigest: "source-a", RuntimeDigest: "runtime-a", ProviderDigest: "provider-a",
		PolicyDigest: "policy-a", ContentDigest: "content-a",
	}
	first, err := NewContextIdentity(base)
	require.NoError(t, err)
	second, err := NewContextIdentity(base)
	require.NoError(t, err)
	require.Equal(t, first.Key, second.Key)

	for _, mutate := range []func(*ContextIdentityInput){
		func(input *ContextIdentityInput) { input.SourceDigest = "source-b" },
		func(input *ContextIdentityInput) { input.RuntimeDigest = "runtime-b" },
		func(input *ContextIdentityInput) { input.ProviderDigest = "provider-b" },
		func(input *ContextIdentityInput) { input.PolicyDigest = "policy-b" },
		func(input *ContextIdentityInput) { input.ContentDigest = "content-b" },
	} {
		changed := base
		mutate(&changed)
		identity, err := NewContextIdentity(changed)
		require.NoError(t, err)
		require.NotEqual(t, first.Key, identity.Key)
	}
}

func TestNewContextIdentityRejectsMissingDigest(t *testing.T) {
	t.Parallel()
	_, err := NewContextIdentity(ContextIdentityInput{SourceDigest: "source"})
	require.ErrorContains(t, err, "runtime_digest")
}
