package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
)

// Response is the raw provider body and how many attempts produced it. The
// attempt count is set on failure too, so a caller can spend its call budget.
type Response struct {
	Body     []byte
	Attempts int
}

// Post sends body to the endpoint with the secret as a bearer token.
func (c *Client) Post(ctx context.Context, correlation integration.Correlation, secret credential.Secret, body []byte) (Response, error) {
	if int64(len(body)) > c.cfg.MaxRequestBytes {
		return Response{}, integration.NewError(integration.StateDataPolicyDenied, correlation, "request exceeds the local size limit")
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return Response{}, integration.NewError(integration.StateRateLimited, correlation, "local concurrency limit reached")
	}
	return c.attempts(ctx, correlation, secret, body)
}

func (c *Client) attempts(ctx context.Context, correlation integration.Correlation, secret credential.Secret, body []byte) (Response, error) {
	var response Response
	for attempt := 1; ; attempt++ {
		var err error
		response.Attempts = attempt
		response.Body, err = c.once(ctx, correlation, secret, body)
		if err == nil || attempt > c.cfg.MaxRetries || !retryable(err) {
			return response, err
		}
	}
}

func retryable(err error) bool {
	state, ok := integration.StateOf(err)
	return ok && state == integration.StateUnavailable
}

func (c *Client) once(ctx context.Context, correlation integration.Correlation, secret credential.Secret, body []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, integration.NewError(integration.StateUnavailable, correlation, "request could not be built")
	}
	request.Header.Set("Authorization", "Bearer "+secret.Reveal())
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, networkError(correlation, err)
	}
	defer closeQuietly(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, statusError(correlation, response.StatusCode)
	}
	return readLimited(correlation, response.Body, c.cfg.MaxResponseBytes)
}

func readLimited(correlation integration.Correlation, body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, networkError(correlation, err)
	}
	if int64(len(data)) > limit {
		return nil, integration.NewError(integration.StateInvalidResponse, correlation, "response exceeds the local size limit")
	}
	return data, nil
}

// closeQuietly closes a response body after it was read to its limit; a close
// error there has no recovery and no bearing on the result.
func closeQuietly(body io.Closer) {
	if err := body.Close(); err != nil {
		return
	}
}
