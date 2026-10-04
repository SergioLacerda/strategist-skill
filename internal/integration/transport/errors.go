package transport

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// networkError classifies a transport failure. Reasons are fixed text: the
// underlying error can carry the URL, so it is never copied.
func networkError(correlation integration.Correlation, err error) error {
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return integration.NewError(integration.StateTimeout, correlation, "provider did not answer in time")
	}
	return integration.NewError(integration.StateUnavailable, correlation, "provider is unreachable")
}

// statusError maps an HTTP status to a typed state without echoing the body.
func statusError(correlation integration.Correlation, status int) error {
	state, reason := integration.StateInvalidResponse, "provider returned an unexpected status"
	switch {
	case status == http.StatusUnauthorized:
		state, reason = integration.StateAuthenticationFailed, "provider rejected the credential"
	case status == http.StatusForbidden:
		state, reason = integration.StateForbidden, "provider denied the operation"
	case status == http.StatusTooManyRequests:
		state, reason = integration.StateRateLimited, "provider rate limit reached"
	case status == http.StatusRequestTimeout || status == http.StatusGatewayTimeout:
		state, reason = integration.StateTimeout, "provider timed out"
	case status >= http.StatusInternalServerError:
		state, reason = integration.StateUnavailable, "provider is unavailable"
	case status >= http.StatusMultipleChoices && status < http.StatusBadRequest:
		reason = "provider redirect was denied"
	}
	return integration.NewError(state, correlation, reason)
}
