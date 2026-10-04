package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/stretchr/testify/require"
)

const syntheticKey = "sk-synthetic-key"

func secret(t *testing.T) credential.Secret {
	t.Helper()
	got, err := credential.Resolve("env:K", credential.Env{Getenv: func(string) string { return syntheticKey }})
	require.NoError(t, err)
	return got
}

func newClient(t *testing.T, server *httptest.Server, mutate func(*Config)) *Client {
	t.Helper()
	cfg := Config{Endpoint: server.URL + "/v1/systemone", RoundTripper: server.Client().Transport}
	if mutate != nil {
		mutate(&cfg)
	}
	client, err := New(cfg)
	require.NoError(t, err)
	return client
}

func stateOf(t *testing.T, err error) integration.State {
	t.Helper()
	state, ok := integration.StateOf(err)
	require.True(t, ok, "expected an integration error, got %v", err)
	return state
}

func TestPostSendsBearerJSONAndReturnsBody(t *testing.T) {
	var gotAuth, gotType, gotBody string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotType = r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	response, err := newClient(t, server, nil).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{"q":1}`))
	require.NoError(t, err)
	require.Equal(t, "Bearer "+syntheticKey, gotAuth)
	require.Equal(t, "application/json", gotType)
	require.Equal(t, `{"q":1}`, gotBody)
	require.JSONEq(t, `{"ok":true}`, string(response.Body))
	require.Equal(t, 1, response.Attempts)
}

func TestNewRejectsEndpointsThatAreNotPlainHTTPS(t *testing.T) {
	for name, endpoint := range map[string]string{
		"http":      "http://api.example.test/v1",
		"userinfo":  "https://user:pass@api.example.test/v1",
		"no host":   "https:///v1",
		"fragment":  "https://api.example.test/v1#x",
		"not a url": "::::",
		"empty":     "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(Config{Endpoint: endpoint})
			require.Error(t, err)
		})
	}
}

func TestStatusCodesMapToTypedStates(t *testing.T) {
	for status, want := range map[int]integration.State{
		401: integration.StateAuthenticationFailed,
		403: integration.StateForbidden,
		429: integration.StateRateLimited,
		408: integration.StateTimeout,
		504: integration.StateTimeout,
		500: integration.StateUnavailable,
		503: integration.StateUnavailable,
		400: integration.StateInvalidResponse,
		404: integration.StateInvalidResponse,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("provider body " + syntheticKey))
			}))
			defer server.Close()

			_, err := newClient(t, server, func(c *Config) { c.MaxRetries = -1 }).Post(context.Background(), integration.Correlation{MissionID: "m"}, secret(t), []byte(`{}`))
			require.Equal(t, want, stateOf(t, err))
			require.NotContains(t, err.Error(), syntheticKey, "neither the key nor the provider body reaches the error")
			require.NotContains(t, err.Error(), "provider body")
		})
	}
}

func TestRedirectsAreDeniedAndNeverFollowed(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed.Add(1) }))
	defer target.Close()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()

	_, err := newClient(t, server, nil).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Equal(t, integration.StateInvalidResponse, stateOf(t, err))
	require.Zero(t, followed.Load())
}

func TestSizeLimitsAreEnforcedLocally(t *testing.T) {
	var reached atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached.Add(1)
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer server.Close()
	client := newClient(t, server, func(c *Config) { c.MaxRequestBytes, c.MaxResponseBytes = 10, 50 })

	_, err := client.Post(context.Background(), integration.Correlation{}, secret(t), []byte(strings.Repeat("a", 11)))
	require.Equal(t, integration.StateDataPolicyDenied, stateOf(t, err))
	require.Zero(t, reached.Load(), "an oversized request is never sent")

	_, err = client.Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Equal(t, integration.StateInvalidResponse, stateOf(t, err))
}

func TestTimeoutMapsToTimeoutState(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer server.Close()

	_, err := newClient(t, server, func(c *Config) { c.Timeout = 50 * time.Millisecond; c.MaxRetries = -1 }).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Equal(t, integration.StateTimeout, stateOf(t, err))
}

func TestRetryIsSingleAndCountedAndOnlyForUnavailable(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	response, err := newClient(t, server, nil).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, 2, response.Attempts)

	var always atomic.Int32
	failing := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		always.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()
	response, err = newClient(t, failing, func(c *Config) { c.MaxRetries = 5 }).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Error(t, err)
	require.EqualValues(t, 2, always.Load(), "the retry ceiling is one even when configured higher")
	require.Equal(t, 2, response.Attempts)

	var denied atomic.Int32
	unauthorized := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		denied.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()
	_, err = newClient(t, unauthorized, nil).Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Error(t, err)
	require.EqualValues(t, 1, denied.Load(), "an authentication failure is never retried")
}

func TestConcurrencyLimitFailsFastAsRateLimited(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := newClient(t, server, func(c *Config) { c.MaxConcurrent = 1 })

	key := secret(t)
	done := make(chan error, 1)
	go func() {
		_, err := client.Post(context.Background(), integration.Correlation{}, key, []byte(`{}`))
		done <- err
	}()
	<-entered

	_, err := client.Post(context.Background(), integration.Correlation{}, secret(t), []byte(`{}`))
	require.Equal(t, integration.StateRateLimited, stateOf(t, err))
	close(release)
	require.NoError(t, <-done)
}
