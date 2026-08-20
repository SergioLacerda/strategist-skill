package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// eventJSON is the JSON-line shape for an appended Event — a flat structure
// so sink/jsonl output stays greppable, mirroring OutcomeEntry's own style.
type eventJSON struct {
	Name           string         `json:"name"`
	Timestamp      string         `json:"timestamp"`
	SeverityNumber int            `json:"severity_number"`
	SeverityText   string         `json:"severity_text"`
	Body           string         `json:"body,omitempty"`
	Attributes     map[string]any `json:"attributes,omitempty"`
	TraceID        string         `json:"trace_id,omitempty"`
	SpanID         string         `json:"span_id,omitempty"`
}

// MarshalEventLine renders event as a single JSON line, suitable for
// AppendEventLine or any other line-oriented sink.
func MarshalEventLine(event Event) (string, error) {
	if err := event.Validate(); err != nil {
		return "", err
	}
	line, err := json.Marshal(eventJSON{
		Name:           event.Name,
		Timestamp:      event.Timestamp.UTC().Format("2006-01-02T15:04:05.000000000Z"),
		SeverityNumber: int(event.SeverityNumber),
		SeverityText:   event.SeverityNumber.String(),
		Body:           event.Body,
		Attributes:     event.Attributes,
		TraceID:        event.TraceID,
		SpanID:         event.SpanID,
	})
	if err != nil {
		return "", fmt.Errorf("event: marshal: %w", err)
	}
	return string(line), nil
}

// AppendEventLine appends event as one JSON line to path, creating the file
// if absent. Unlike AppendOutcomeLine, events are an append-only stream, not
// a mission_id-keyed set — there is no dedup key, every call appends. Uses
// the same shared-flock append discipline as AppendOutcomeLine so a
// jsonl sink writer is safe to run concurrently with other Strategist
// processes appending to the same file.
func AppendEventLine(path string, event Event) (err error) {
	line, err := MarshalEventLine(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // G304: event log path is owned by the Strategist runtime memory domain
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer closeFileWithContext(f, &err, "close event log")

	if err = lockFile(f); err != nil {
		return fmt.Errorf("lock event log: %w", err)
	}
	defer func() {
		if unlockErr := unlockFile(f); unlockErr != nil && err == nil {
			err = fmt.Errorf("unlock event log: %w", unlockErr)
		}
	}()

	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek event log: %w", err)
	}
	if _, err = fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("write event log line: %w", err)
	}
	return nil
}
