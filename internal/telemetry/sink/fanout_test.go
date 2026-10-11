package sink_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry/sink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSink struct {
	events []telemetry.Event
	err    error
}

func (r *recordingSink) Emit(_ context.Context, e telemetry.Event) error {
	r.events = append(r.events, e)
	return r.err
}

func TestFanoutDeliversToEverySinkEvenWhenOneFails(t *testing.T) {
	failing := &recordingSink{err: errors.New("collector down")}
	healthy := &recordingSink{}
	err := sink.Fanout(failing, healthy).Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now()})

	require.Error(t, err)
	assert.Len(t, failing.events, 1)
	assert.Len(t, healthy.events, 1, "a failing sink must not starve the others")
}

func TestSelectWithAnEventsFileWritesEveryEventAsAJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	s := sink.Select(telemetry.Config{EventsFile: path}, nil)
	require.NoError(t, s.Emit(context.Background(), telemetry.Event{Name: "strategist.test", Timestamp: time.Now(), Complete: true}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"name":"strategist.test"`)
}

func TestSelectWithoutAnEventsFileDoesNotTouchTheFilesystem(t *testing.T) {
	dir := t.TempDir()
	s := sink.Select(telemetry.Config{}, nil)
	require.NoError(t, s.Emit(context.Background(), telemetry.Event{Name: "strategist.test", Timestamp: time.Now(), Complete: true}))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSelectStrictPropagatesAnUnwritableEventsFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	path := filepath.Join(blocker, "events.jsonl") // parent is a regular file

	strict := sink.Select(telemetry.Config{EventsFile: path, Strict: true}, nil)
	require.Error(t, strict.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now(), Complete: true}))

	lenient := sink.Select(telemetry.Config{EventsFile: path}, nil)
	require.NoError(t, lenient.Emit(context.Background(), telemetry.Event{Name: "x", Timestamp: time.Now(), Complete: true}),
		"non-strict export failure never blocks the caller")
}
