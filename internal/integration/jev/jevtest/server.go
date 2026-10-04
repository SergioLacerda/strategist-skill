// Package jevtest is a simulated JEV provider for tests: a TLS server that
// records what it receives and replies with whatever the test supplies. It never
// contacts the real service.
package jevtest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// Server is a recording simulated provider.
type Server struct {
	server *httptest.Server
	calls  atomic.Int32
	mu     sync.Mutex
	auth   string
	body   string
}

// Start serves handler over TLS and records each request.
func Start(t *testing.T, handler http.HandlerFunc) *Server {
	t.Helper()
	sim := &Server{}
	sim.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unreadable body", http.StatusBadRequest)
			return
		}
		sim.mu.Lock()
		sim.auth, sim.body = r.Header.Get("Authorization"), string(raw)
		sim.mu.Unlock()
		sim.calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(sim.server.Close)
	return sim
}

// Reply answers every request with a 200 and body ("{}" when nil).
func Reply(body []byte) http.HandlerFunc {
	if body == nil {
		body = []byte("{}")
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body); err != nil {
			return
		}
	}
}

// Status answers every request with the given status and no body.
func Status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }
}

// URL is the base URL of the simulator.
func (s *Server) URL() string { return s.server.URL }

// Transport trusts the simulator's certificate.
func (s *Server) Transport() http.RoundTripper { return s.server.Client().Transport }

// Calls counts the requests received.
func (s *Server) Calls() int { return int(s.calls.Load()) }

// LastAuthorization is the Authorization header of the latest request.
func (s *Server) LastAuthorization() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auth
}

// LastBody is the body of the latest request.
func (s *Server) LastBody() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.body
}
