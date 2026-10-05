// Package jevtest is a simulated JEV provider for tests: an HTTPS-shaped,
// in-memory transport that records what it receives and replies with whatever
// the test supplies. It never opens a socket or contacts the real service.
package jevtest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// Server is a recording simulated provider.
type Server struct {
	handler http.Handler
	url     string
	host    string
	calls   atomic.Int32
	mu      sync.Mutex
	auth    string
	body    string
}

var serverSequence atomic.Uint64

// Start serves handler through an in-memory HTTPS-shaped transport and records
// each request.
func Start(t *testing.T, handler http.HandlerFunc) *Server {
	t.Helper()
	host := fmt.Sprintf("jevtest-%d.invalid", serverSequence.Add(1))
	sim := &Server{url: "https://" + host, host: host}
	sim.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unreadable body", http.StatusBadRequest)
			return
		}
		sim.mu.Lock()
		sim.auth, sim.body = r.Header.Get("Authorization"), string(raw)
		sim.mu.Unlock()
		sim.calls.Add(1)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		handler(w, r)
	})
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
func (s *Server) URL() string { return s.url }

// Transport dispatches requests to the simulator in memory. It never opens a
// socket and therefore cannot reach a real provider or depend on IPv4/IPv6.
func (s *Server) Transport() http.RoundTripper { return inMemoryTransport{server: s} }

type inMemoryTransport struct{ server *Server }

func (t inMemoryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != "https" || request.URL.Host != t.server.host {
		return nil, fmt.Errorf("jevtest: unsupported synthetic URL %q", request.URL.String())
	}
	responses := make(chan *http.Response, 1)
	go func() {
		recorder := httptest.NewRecorder()
		t.server.handler.ServeHTTP(recorder, request)
		response := recorder.Result()
		response.Request = request
		responses <- response
	}()
	select {
	case response := <-responses:
		return response, nil
	case <-request.Context().Done():
		return nil, fmt.Errorf("jevtest: request canceled: %w", request.Context().Err())
	}
}

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
