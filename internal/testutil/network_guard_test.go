package testutil

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScrubProviderEnvUnsetsEveryCredentialVariable(t *testing.T) {
	for _, name := range ProviderEnvVars {
		t.Setenv(name, "sk-from-the-developer-shell")
	}
	ScrubProviderEnv()
	for _, name := range ProviderEnvVars {
		require.Empty(t, os.Getenv(name), name)
	}
}

func TestDenyNetworkRefusesAndCountsEveryRequest(t *testing.T) {
	guard := NewDenyNetwork()
	client := &http.Client{Transport: guard}

	_, err := client.Get("https://api.typesafe.ai/v1/systemone") //nolint:bodyclose,noctx // the guard never returns a response
	require.ErrorIs(t, err, ErrNetworkDenied)
	_, err = client.Get("https://example.test/") //nolint:bodyclose,noctx // the guard never returns a response
	require.ErrorIs(t, err, ErrNetworkDenied)
	require.Equal(t, 2, guard.Attempts())

	require.Zero(t, DenyNetwork{}.Attempts(), "a zero guard still refuses and reports zero")
	_, err = DenyNetwork{}.RoundTrip(nil)
	require.ErrorIs(t, err, ErrNetworkDenied)
}
