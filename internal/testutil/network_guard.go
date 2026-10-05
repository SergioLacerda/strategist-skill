package testutil

import (
	"errors"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
)

// ProviderEnvVars are the environment variables that carry a provider
// credential. A test run must never inherit them from the developer's shell, so
// a test cannot pass or fail because a real key happens to be exported.
var ProviderEnvVars = []string{"TYPESAFE_API_KEY", "JEV_API_KEY"}

// ScrubProviderEnv unsets every provider credential variable of the process.
func ScrubProviderEnv() {
	for _, name := range ProviderEnvVars {
		_ = os.Unsetenv(name) //nolint:errcheck // unsetting a variable cannot fail in a way a test could act on
	}
}

// RunHermetic scrubs the provider credentials and then runs the package under
// the home guard. Use it as a package's TestMain.
func RunHermetic(m *testing.M) int {
	ScrubProviderEnv()
	return RunWithHomeGuard(m)
}

// ErrNetworkDenied is returned by DenyNetwork for every request.
var ErrNetworkDenied = errors.New("network is denied in tests: inject a simulated provider, never the real one")

// DenyNetwork is an http.RoundTripper that refuses every request and counts the
// attempts. Installed as the default transport of a test binary, it turns a test
// that forgot to inject a simulator into a loud failure instead of a real call.
type DenyNetwork struct{ attempts *atomic.Int32 }

// NewDenyNetwork builds a guard with its own attempt counter.
func NewDenyNetwork() DenyNetwork { return DenyNetwork{attempts: &atomic.Int32{}} }

// RoundTrip refuses the request.
func (d DenyNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	if d.attempts != nil {
		d.attempts.Add(1)
	}
	return nil, ErrNetworkDenied
}

// Attempts reports how many requests the guard refused.
func (d DenyNetwork) Attempts() int {
	if d.attempts == nil {
		return 0
	}
	return int(d.attempts.Load())
}
