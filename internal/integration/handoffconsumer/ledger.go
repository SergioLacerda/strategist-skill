package handoffconsumer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Record is one line of the integration call ledger: metadata about a pre-check,
// never artifact content, a provider body or a credential. It is the call budget's
// source of truth and the ground truth the pilot measures against.
type Record struct {
	At            string  `json:"at"`
	MissionID     string  `json:"mission_id"`
	Transition    string  `json:"transition"`
	Subject       string  `json:"subject"`
	PreferredPath string  `json:"preferred_path"`
	EffectivePath string  `json:"effective_path"`
	Reason        string  `json:"reason,omitempty"`
	FellBack      bool    `json:"fell_back"`
	Suspect       bool    `json:"suspect,omitempty"`
	Binding       string  `json:"binding"`
	Model         string  `json:"model,omitempty"`
	Attempted     bool    `json:"attempted"`
	Approved      bool    `json:"approved"`
	Confidence    float64 `json:"confidence,omitempty"`
	Threshold     float64 `json:"threshold,omitempty"`
	InputTokens   int     `json:"input_tokens,omitempty"`
	OutputTokens  int     `json:"output_tokens,omitempty"`
	LatencyMS     int64   `json:"latency_ms,omitempty"`
}

// Ledger is an append-only JSONL file.
type Ledger struct{ Path string }

// Append writes one record with owner-only permissions.
func (l Ledger) Append(record Record) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode ledger record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o750); err != nil {
		return fmt.Errorf("create ledger directory: %w", err)
	}
	file, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: ledger path is owned by runtime memory
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return errors.Join(fmt.Errorf("write ledger: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close ledger: %w", err)
	}
	return nil
}

// Count returns how many provider calls were attempted for a mission and
// transition. A missing ledger is zero; malformed lines are ignored.
func (l Ledger) Count(missionID, transition string) (int, error) {
	file, err := os.Open(l.Path) //nolint:gosec // G304: ledger path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("open ledger: %w", err)
	}
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) == nil && record.Attempted && record.MissionID == missionID && record.Transition == transition {
			count++
		}
	}
	return count, errors.Join(scanner.Err(), file.Close())
}
