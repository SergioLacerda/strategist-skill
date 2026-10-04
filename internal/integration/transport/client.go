// Package transport is the HTTPS client of the Integration Mechanism. It talks
// only to the one operator-configured endpoint, never follows a redirect,
// ignores proxy environment variables, enforces local size and concurrency
// limits and maps every failure to a typed integration state.
package transport

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout       = 30 * time.Second
	defaultRequestBytes  = 64 << 10
	defaultResponseBytes = 256 << 10
	defaultConcurrent    = 2
	retryCeiling         = 1
)

// Config holds the operator-controlled transport parameters. Zero values take
// conservative defaults. RoundTripper is a test seam; production leaves it nil.
type Config struct {
	Endpoint         string
	Timeout          time.Duration
	MaxRequestBytes  int64
	MaxResponseBytes int64
	MaxRetries       int // 0 = one retry (default), negative = none, capped at one
	MaxConcurrent    int
	RoundTripper     http.RoundTripper
}

// Client posts JSON to the configured endpoint.
type Client struct {
	endpoint string
	cfg      Config
	http     *http.Client
	slots    chan struct{}
}

// New validates the endpoint and builds a client.
func New(cfg Config) (*Client, error) {
	if err := validateEndpoint(cfg.Endpoint); err != nil {
		return nil, err
	}
	cfg = withDefaults(cfg)
	return &Client{
		endpoint: cfg.Endpoint,
		cfg:      cfg,
		http: &http.Client{
			Transport:     roundTripper(cfg),
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		slots: make(chan struct{}, cfg.MaxConcurrent),
	}, nil
}

func withDefaults(cfg Config) Config {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxRequestBytes <= 0 {
		cfg.MaxRequestBytes = defaultRequestBytes
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultResponseBytes
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = defaultConcurrent
	}
	cfg.MaxRetries = retries(cfg.MaxRetries)
	return cfg
}

// retries applies the single-retry default: zero asks for the default, a
// negative value disables retries and nothing exceeds the ceiling.
func retries(configured int) int {
	switch {
	case configured == 0:
		return retryCeiling
	case configured < 0:
		return 0
	default:
		return min(configured, retryCeiling)
	}
}

// roundTripper disables proxy discovery: an environment proxy could route the
// credential around the endpoint allowlist.
func roundTripper(cfg Config) http.RoundTripper {
	if cfg.RoundTripper != nil {
		return cfg.RoundTripper
	}
	return &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
}

func validateEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	switch {
	case err != nil:
		return errors.New("integration endpoint is not a valid URL")
	case parsed.Scheme != "https":
		return errors.New("integration endpoint must use https")
	case parsed.Host == "" || parsed.User != nil || parsed.Fragment != "":
		return errors.New("integration endpoint must be a plain https URL with a host")
	}
	return nil
}
