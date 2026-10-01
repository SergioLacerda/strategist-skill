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
	if err := requireCommittableAdapter(record.ExecutionAdapter); err != nil {
		return err
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
	s.prune(filepath.Dir(path))
	return atomicWriteJSON(path, record)
}

// requireCommittableAdapter accepts an absent adapter (a pre-field record) or a
// mode Strategist may commit; anything else is outside the closed vocabulary.
func requireCommittableAdapter(adapter domain.MissionExecutionAdapter) error {
	if adapter != "" && !adapter.Committable() {
		return fmt.Errorf("invocation_adapter_unknown: %q is not a committable execution adapter", adapter)
	}
	return nil
}

// Get loads a request that can still be completed: pending or processing. It
// rejects a completed request as a replay, and a pending one past its expiry.
// A processing request is never expired out from under its recovery.
func (s InvocationStore) Get(requestID string) (domain.MissionInvocationRecord, error) {
	record, err := s.load(requestID)
	if err != nil {
		return domain.MissionInvocationRecord{}, err
	}
	switch record.EffectiveState() {
	case domain.InvocationStateCompleted:
		return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_replay: request %q was already consumed", requestID)
	case domain.InvocationStatePending:
		if !record.ExpiresAt.IsZero() && s.now().After(record.ExpiresAt) {
			return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_request_expired: request %q expired", requestID)
		}
	case domain.InvocationStateProcessing:
		// Never expired: a committed completion must stay recoverable.
	}
	return record, nil
}

func (s InvocationStore) load(requestID string) (domain.MissionInvocationRecord, error) {
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
	return record, nil
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
	if err := WriteFileAtomic(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("mission invocation: persist request: %w", err)
	}
	return nil
}
