package jevtest

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartUsesAnInMemoryHTTPSTransport(t *testing.T) {
	sim := Start(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}
		assert.JSONEq(t, `{"hello":"world"}`, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	assert.Regexp(t, `^https://jevtest-[0-9]+\.invalid$`, sim.URL())
	request, err := http.NewRequest(http.MethodPost, sim.URL()+"/v1/test", strings.NewReader(`{"hello":"world"}`))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer test-key")
	response, err := (&http.Client{Transport: sim.Transport()}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, response.StatusCode)
	assert.Equal(t, `{"ok":true}`, string(body))
	assert.Equal(t, 1, sim.Calls())
	assert.Equal(t, "Bearer test-key", sim.LastAuthorization())
	assert.JSONEq(t, `{"hello":"world"}`, sim.LastBody())
}
