package leveling

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// BudgetLedgerSchemaVersion identifies the isolated effort-event stream. It
// is intentionally not LedgerSchemaVersion: role-level readers must not ingest
// budget decisions.
const BudgetLedgerSchemaVersion = "strategist-effort-budget-event/v1"

// BudgetEventKind is the closed vocabulary for budget audit events.
type BudgetEventKind string

const (
	// BudgetEventRecommendation records a policy recommendation.
	BudgetEventRecommendation BudgetEventKind = "recommendation"
	// BudgetEventGrant records a local admission.
	BudgetEventGrant BudgetEventKind = "grant"
	// BudgetEventConsumption records completed local use.
	BudgetEventConsumption BudgetEventKind = "consumption"
	// BudgetEventDenial records a local rejection.
	BudgetEventDenial BudgetEventKind = "denial"
	// BudgetEventExhaustion records capacity exhaustion.
	BudgetEventExhaustion BudgetEventKind = "exhaustion"
	// BudgetEventEscalation records a bounded escalation.
	BudgetEventEscalation BudgetEventKind = "escalation"
	// BudgetEventProviderUsage records provider-usage evidence.
	BudgetEventProviderUsage BudgetEventKind = "provider_usage"
)

// BudgetEventRecord is one immutable decision in the isolated budget stream.
type BudgetEventRecord struct {
	SchemaVersion      string          `json:"schema_version"`
	Timestamp          string          `json:"timestamp"`
	Event              BudgetEventKind `json:"event"`
	MissionID          string          `json:"mission_id"`
	RunID              string          `json:"run_id"`
	Role               string          `json:"role"`
	Provider           string          `json:"provider,omitempty"`
	Model              string          `json:"model,omitempty"`
	Capability         string          `json:"capability,omitempty"`
	Reason             string          `json:"reason,omitempty"`
	ProviderEvidence   string          `json:"provider_evidence,omitempty"`
	ResolutionID       string          `json:"resolution_id,omitempty"`
	ParentResolutionID string          `json:"parent_resolution_id,omitempty"`
	PolicyVersion      int             `json:"policy_version,omitempty"`
	PolicyDigest       string          `json:"policy_digest,omitempty"`
	Remaining          int             `json:"remaining_delegations"`
	Active             int             `json:"active_delegations"`
}

// NewBudgetEventRecord creates a correlated event from a resolved budget.
func NewBudgetEventRecord(budget domain.EffortBudget, event BudgetEventKind, reason, providerEvidence string, remaining, active int) BudgetEventRecord {
	return BudgetEventRecord{
		SchemaVersion: BudgetLedgerSchemaVersion, Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Event: event,
		MissionID: budget.MissionID, RunID: budget.RunID, Role: NormalizeRole(budget.Role), Provider: budget.Provider,
		Model: budget.Model, Capability: budget.Capability, Reason: reason, ProviderEvidence: providerEvidence,
		ResolutionID: budget.ResolutionID, ParentResolutionID: budget.ParentResolutionID,
		PolicyVersion: budget.PolicyVersion, PolicyDigest: budget.PolicyDigest, Remaining: remaining, Active: active,
	}
}

// AppendBudgetEvent appends one validated event with the same lock and file
// permission guarantees as the role-level ledger.
func AppendBudgetEvent(path string, record BudgetEventRecord) error {
	if record.SchemaVersion == "" {
		record.SchemaVersion = BudgetLedgerSchemaVersion
	}
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	record.Role = NormalizeRole(record.Role)
	if err := validateBudgetEvent(record); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("leveling: create budget ledger directory: %w", err)
	}
	return withLedgerLock(path, func() error {
		encoded, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("leveling: encode budget event: %w", err)
		}
		return appendJSONLine(path, encoded, "budget")
	})
}

// ReadBudgetEvents reads valid events in append order. Missing or malformed
// history is non-blocking, matching the role-level ledger read contract.
func ReadBudgetEvents(path string) ([]BudgetEventRecord, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("leveling: create budget ledger directory: %w", err)
	}
	var records []BudgetEventRecord
	err := withLedgerLock(path, func() error {
		var readErr error
		records, readErr = readBudgetEventsUnlocked(path)
		return readErr
	})
	return records, err
}

func readBudgetEventsUnlocked(path string) ([]BudgetEventRecord, error) {
	f, err := os.Open(path) //nolint:gosec // path is resolved below .strategist
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leveling: open budget ledger: %w", err)
	}
	defer f.Close() //nolint:errcheck // scanner result is the meaningful read error
	return scanBudgetEvents(f)
}

func scanBudgetEvents(f *os.File) ([]BudgetEventRecord, error) {
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxLedgerRecordBytes+1)
	var records []BudgetEventRecord
	for scanner.Scan() {
		var record BudgetEventRecord
		if json.Unmarshal(scanner.Bytes(), &record) == nil && validateBudgetEvent(record) == nil {
			record.Role = NormalizeRole(record.Role)
			records = append(records, record)
		}
	}
	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "token too long") {
			return nil, fmt.Errorf("%s: maximum record size is %d bytes", ReasonLevelingLedgerRecordOversized, maxLedgerRecordBytes)
		}
		return nil, fmt.Errorf("leveling: read budget ledger: %w", err)
	}
	return records, nil
}

func validateBudgetEvent(record BudgetEventRecord) error {
	if record.SchemaVersion != BudgetLedgerSchemaVersion {
		return fmt.Errorf("leveling: unsupported budget event schema %q", record.SchemaVersion)
	}
	if record.MissionID == "" || record.RunID == "" || record.Role == "" {
		return fmt.Errorf("leveling: budget event identity is required")
	}
	for _, allowed := range []BudgetEventKind{BudgetEventRecommendation, BudgetEventGrant, BudgetEventConsumption, BudgetEventDenial, BudgetEventExhaustion, BudgetEventEscalation, BudgetEventProviderUsage} {
		if record.Event == allowed {
			return nil
		}
	}
	return fmt.Errorf("leveling: unsupported budget event %q", record.Event)
}
