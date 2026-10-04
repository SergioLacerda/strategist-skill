package providerapp_test

import (
	"errors"
	"testing"

	providerapp "github.com/SergioLacerda/strategist-skill/internal/application/provider"
	"github.com/stretchr/testify/require"
)

func TestValidateProviderDelegatesToThePort(t *testing.T) {
	want := providerapp.ProviderReport{ProviderID: "fixture"}
	got, err := providerapp.ValidateProvider("source", "", providerapp.ProviderPorts{
		Validate: func(source, slot string) (providerapp.ProviderReport, error) {
			require.Equal(t, "source", source)
			require.Empty(t, slot)
			return want, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestValidateProviderRejectsInvalidPorts(t *testing.T) {
	_, err := providerapp.ValidateProvider("", "", providerapp.ProviderPorts{})
	require.ErrorContains(t, err, "source is required")
	_, err = providerapp.ValidateProvider("source", "", providerapp.ProviderPorts{})
	require.ErrorContains(t, err, "adapter is unavailable")
}

func TestAddProviderDelegatesAndPreservesAdapterError(t *testing.T) {
	wantErr := errors.New("stage failed")
	got, err := providerapp.AddProvider("root", "source", "discovery", providerapp.ProviderPorts{
		Add: func(root, source, slot string) (providerapp.ProviderAddResult, error) {
			require.Equal(t, "root", root)
			require.Equal(t, "source", source)
			require.Equal(t, "discovery", slot)
			return providerapp.ProviderAddResult{}, wantErr
		},
	})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, providerapp.ProviderAddResult{}, got)
}

func TestAddProviderRejectsInvalidPorts(t *testing.T) {
	for _, tc := range []struct {
		name, root, source, slot, reason string
	}{
		{"root", "", "source", "slot", "strategist root"},
		{"source", "root", "", "slot", "provider source"},
		{"slot", "root", "source", "", "provider slot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := providerapp.AddProvider(tc.root, tc.source, tc.slot, providerapp.ProviderPorts{})
			require.ErrorContains(t, err, tc.reason)
		})
	}
	_, err := providerapp.AddProvider("root", "source", "slot", providerapp.ProviderPorts{})
	require.ErrorContains(t, err, "onboarding adapter is unavailable")
}
