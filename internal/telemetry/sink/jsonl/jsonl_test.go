package jsonlsink_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	jsonlsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/jsonl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSink_Emit_AppendsJSONLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	s := jsonlsink.New(path)

	err := s.Emit(context.Background(), telemetry.Event{
		Name:      "strategist.governance.decision",
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	data, err := os.ReadFile(path) //nolint:gosec // G304: test temp path
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data[:len(data)-1], &decoded)) // strip trailing newline
	assert.Equal(t, "strategist.governance.decision", decoded["name"])
}

func TestSink_Emit_InvalidEventPropagatesError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := jsonlsink.New(filepath.Join(dir, "events.jsonl"))
	err := s.Emit(context.Background(), telemetry.Event{})
	require.Error(t, err)
}
