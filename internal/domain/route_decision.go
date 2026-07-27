package domain

import "fmt"

// RouteValidationStatus reports the outcome of route decision validation.
type RouteValidationStatus string

// Route decision validation statuses.
const (
	RouteValidationAllowed RouteValidationStatus = "allowed"
	RouteValidationBlocked RouteValidationStatus = "blocked"
)

// RouteRequestMetadata captures optional request facts used to validate a route decision.
type RouteRequestMetadata struct {
	// HasContext means the caller supplied request facts. Without context, validation is
	// intentionally limited to checking that the route name is known.
	HasContext bool
	// TouchesSourceCode indicates the request mutates source files or tests.
	TouchesSourceCode bool
	// AnalysisArtifactOnly indicates the request only moves/archives/reopens analysis files.
	AnalysisArtifactOnly bool
	// RequiresDiscovery indicates the request needs Ranger/Archivist analysis before execution.
	RequiresDiscovery bool
}

// RouteValidationDecision is the deterministic result of validating a route decision.
type RouteValidationDecision struct {
	Allowed bool
	Status  RouteValidationStatus
	Route   string
	Reason  string
}

// ValidateRouteDecision checks whether Scout's selected route is compatible with known
// request metadata. It is conservative when metadata is absent so existing callers can adopt
// the hook before they have full request classification signals.
func ValidateRouteDecision(route string, metadata RouteRequestMetadata) RouteValidationDecision {
	if route == "" {
		route = MissionRouteMain
	}
	switch route {
	case MissionRouteMain:
		return allowedRoute(route)
	case MissionRouteDirectExecute:
		return validateDirectExecuteRoute(route, metadata)
	default:
		return blockedRoute(route, fmt.Sprintf("unknown route %q", route))
	}
}

func validateDirectExecuteRoute(route string, metadata RouteRequestMetadata) RouteValidationDecision {
	if !metadata.HasContext {
		return allowedRoute(route)
	}
	if metadata.TouchesSourceCode {
		return blockedRoute(route, "direct_execute cannot mutate source files or tests")
	}
	if metadata.RequiresDiscovery {
		return blockedRoute(route, "direct_execute cannot bypass required discovery/refinement")
	}
	if !metadata.AnalysisArtifactOnly {
		return blockedRoute(route, "direct_execute is limited to analysis artifact maintenance")
	}
	return allowedRoute(route)
}

func allowedRoute(route string) RouteValidationDecision {
	return RouteValidationDecision{
		Allowed: true,
		Status:  RouteValidationAllowed,
		Route:   route,
	}
}

func blockedRoute(route, reason string) RouteValidationDecision {
	return RouteValidationDecision{
		Allowed: false,
		Status:  RouteValidationBlocked,
		Route:   route,
		Reason:  reason,
	}
}
