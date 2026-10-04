package authorizationapp_test

import (
	"testing"

	authorizationapp "github.com/SergioLacerda/strategist-skill/internal/application/authorization"
	"github.com/stretchr/testify/require"
)

func TestAuthorizeDelegatesAValidatedRequest(t *testing.T) {
	want := authorizationapp.AuthorizationReport{Decision: "allowed"}
	got, err := authorizationapp.Authorize(authorizationapp.AuthorizationRequest{Target: "docs/report.md"}, func(request authorizationapp.AuthorizationRequest) (authorizationapp.AuthorizationReport, error) {
		require.Equal(t, "docs/report.md", request.Target)
		return want, nil
	})
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestAuthorizeRejectsMissingTargetOrAdapter(t *testing.T) {
	_, err := authorizationapp.Authorize(authorizationapp.AuthorizationRequest{}, nil)
	require.ErrorContains(t, err, "target is required")
	_, err = authorizationapp.Authorize(authorizationapp.AuthorizationRequest{Target: "target"}, nil)
	require.ErrorContains(t, err, "adapter is unavailable")
}
