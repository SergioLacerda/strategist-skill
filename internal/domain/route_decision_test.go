package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidateRouteDecision_DefaultsEmptyRouteToMain(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision("", domain.RouteRequestMetadata{})

	assert.True(t, decision.Allowed)
	assert.Equal(t, domain.RouteValidationAllowed, decision.Status)
	assert.Equal(t, domain.MissionRouteMain, decision.Route)
}

func TestValidateRouteDecision_AllowsKnownRoutesWithoutContext(t *testing.T) {
	t.Parallel()

	for _, route := range []string{
		domain.MissionRouteMain,
		domain.MissionRouteQuickDraw,
		domain.MissionRouteDirectExecute,
	} {
		decision := domain.ValidateRouteDecision(route, domain.RouteRequestMetadata{})
		assert.True(t, decision.Allowed, route)
		assert.Equal(t, domain.RouteValidationAllowed, decision.Status, route)
	}
}

func TestValidateRouteDecision_BlocksUnknownRoute(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision("bogus", domain.RouteRequestMetadata{})

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.RouteValidationBlocked, decision.Status)
	assert.Contains(t, decision.Reason, "unknown route")
}

func TestValidateRouteDecision_BlocksDirectExecuteForSourceMutation(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteDirectExecute, domain.RouteRequestMetadata{
		HasContext:        true,
		TouchesSourceCode: true,
	})

	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "source files")
}

func TestValidateRouteDecision_AllowsDirectExecuteForAnalysisArtifactOnly(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteDirectExecute, domain.RouteRequestMetadata{
		HasContext:           true,
		AnalysisArtifactOnly: true,
	})

	assert.True(t, decision.Allowed)
}

func TestValidateRouteDecision_BlocksQuickDrawWhenDiscoveryRequired(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteQuickDraw, domain.RouteRequestMetadata{
		HasContext:        true,
		RequiresDiscovery: true,
	})

	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "discovery")
}

func TestValidateRouteDecision_BlocksQuickDrawForSourceMutation(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteQuickDraw, domain.RouteRequestMetadata{
		HasContext:        true,
		TouchesSourceCode: true,
	})

	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "source files")
}

func TestValidateRouteDecision_AllowsQuickDrawWithContextAndNoRisk(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteQuickDraw, domain.RouteRequestMetadata{
		HasContext: true,
	})

	assert.True(t, decision.Allowed)
}

func TestValidateRouteDecision_BlocksDirectExecuteWhenDiscoveryRequired(t *testing.T) {
	t.Parallel()

	decision := domain.ValidateRouteDecision(domain.MissionRouteDirectExecute, domain.RouteRequestMetadata{
		HasContext:        true,
		RequiresDiscovery: true,
	})

	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "discovery")
}
