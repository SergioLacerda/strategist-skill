package application_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestAuthorizeDelegatesAValidatedRequest(t *testing.T) {
	want := application.AuthorizationReport{Decision: "allowed"}
	got, err := application.Authorize(application.AuthorizationRequest{Target: "docs/report.md"}, func(request application.AuthorizationRequest) (application.AuthorizationReport, error) {
		require.Equal(t, "docs/report.md", request.Target)
		return want, nil
	})
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestAuthorizeRejectsMissingTargetOrAdapter(t *testing.T) {
	_, err := application.Authorize(application.AuthorizationRequest{}, nil)
	require.ErrorContains(t, err, "target is required")
	_, err = application.Authorize(application.AuthorizationRequest{Target: "target"}, nil)
	require.ErrorContains(t, err, "adapter is unavailable")
}
