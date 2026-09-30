package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

var invocationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)

// InvocationStore persists pending host requests below one Strategist root.
// It is intentionally independent from the host agent and uses atomic writes.
type InvocationStore struct {
	Root  string
	Clock func() time.Time
}

// NewInvocationStore creates a file-backed store with a UTC clock.
func NewInvocationStore(root string) InvocationStore {
	return InvocationStore{Root: root, Clock: func() time.Time { return time.Now().UTC() }}
}

// Put records a new request and refuses request-id replacement.
func (s InvocationStore) Put(record domain.MissionInvocationRecord) error {
	if err := record.Request.Validate(); err != nil {
		return fmt.Errorf("mission invocation: validate request: %w", err)
	}
	if !invocationIDPattern.MatchString(record.Request.RequestID) {
		return fmt.Errorf("mission invocation: malformed request_id %q", record.Request.RequestID)
	}
	path, err := s.path(record.Request.RequestID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("invocation_request_replay: request %q already exists", record.Request.RequestID)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("mission invocation: inspect request: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mission invocation: create request directory: %w", err)
	}
	return atomicWriteJSON(path, record)
}

// Get loads a pending request and rejects expired or consumed state.
func (s InvocationStore) Get(requestID string) (domain.MissionInvocationRecord, error) {
	path, err := s.path(requestID)
	if err != nil {
		return domain.MissionInvocationRecord{}, err
	}
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is derived by InvocationStore.path after request-id validation.
	if err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("mission invocation: read request: %w", err)
	}
	var record domain.MissionInvocationRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("mission invocation: parse request: %w", err)
	}
	if record.Consumed {
		return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_replay: request %q was already consumed", requestID)
	}
	if !record.ExpiresAt.IsZero() && s.now().After(record.ExpiresAt) {
		return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_request_expired: request %q expired", requestID)
	}
	return record, nil
}

// Consume marks a request as used after normalization and artifact persistence.
func (s InvocationStore) Consume(requestID string) error {
	record, err := s.Get(requestID)
	if err != nil {
		return err
	}
	now := s.now()
	record.Consumed = true
	record.ConsumedAt = &now
	path, err := s.path(requestID)
	if err != nil {
		return err
	}
	return atomicWriteJSON(path, record)
}

func (s InvocationStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock().UTC()
}

func (s InvocationStore) path(requestID string) (string, error) {
	if !invocationIDPattern.MatchString(requestID) {
		return "", fmt.Errorf("mission invocation: malformed request_id %q", requestID)
	}
	if s.Root == "" {
		return "", fmt.Errorf("mission invocation: root is required")
	}
	return filepath.Join(s.Root, "missions", "invocations", requestID+".json"), nil
}

func atomicWriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("mission invocation: encode request: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".invocation-*.tmp")
	if err != nil {
		return fmt.Errorf("mission invocation: create temporary request: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup of an uncommitted temporary request.
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the chmod error.
		return fmt.Errorf("mission invocation: protect temporary request: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the write error.
		return fmt.Errorf("mission invocation: write temporary request: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("mission invocation: close temporary request: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("mission invocation: persist request: %w", err)
	}
	return nil
}
