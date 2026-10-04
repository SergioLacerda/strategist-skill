package application_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestValidateProviderDelegatesToThePort(t *testing.T) {
	want := application.ProviderReport{ProviderID: "fixture"}
	got, err := application.ValidateProvider("source", "", application.ProviderPorts{
		Validate: func(source, slot string) (application.ProviderReport, error) {
			require.Equal(t, "source", source)
			require.Empty(t, slot)
			return want, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestValidateProviderRejectsInvalidPorts(t *testing.T) {
	_, err := application.ValidateProvider("", "", application.ProviderPorts{})
	require.ErrorContains(t, err, "source is required")
	_, err = application.ValidateProvider("source", "", application.ProviderPorts{})
	require.ErrorContains(t, err, "adapter is unavailable")
}

func TestAddProviderDelegatesAndPreservesAdapterError(t *testing.T) {
	wantErr := errors.New("stage failed")
	got, err := application.AddProvider("root", "source", "discovery", application.ProviderPorts{
		Add: func(root, source, slot string) (application.ProviderAddResult, error) {
			require.Equal(t, "root", root)
			require.Equal(t, "source", source)
			require.Equal(t, "discovery", slot)
			return application.ProviderAddResult{}, wantErr
		},
	})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, application.ProviderAddResult{}, got)
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
			_, err := application.AddProvider(tc.root, tc.source, tc.slot, application.ProviderPorts{})
			require.ErrorContains(t, err, tc.reason)
		})
	}
	_, err := application.AddProvider("root", "source", "slot", application.ProviderPorts{})
	require.ErrorContains(t, err, "onboarding adapter is unavailable")
}
