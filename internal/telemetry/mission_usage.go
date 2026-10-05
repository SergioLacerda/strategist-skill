package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionUsageRelPath is the runtime-root-relative path of the append-only
// mission usage ledger: declared token observations with phase, invocation,
// semantics and quality. The legacy mission-token-usage ledger is left alone.
const MissionUsageRelPath = "memory/mission-usage.jsonl"

// MissionUsageSchemaVersion is the only record schema version.
const MissionUsageSchemaVersion = "1"

// MissionUsageRecord is one JSONL line of the usage ledger. A token member is
// null when it was not observed: unobserved is never stored as zero.
type MissionUsageRecord struct {
	SchemaVersion      string `json:"schema_version"`
	RecordID           string `json:"record_id"`
	EventID            string `json:"event_id,omitempty"`
	MissionID          string `json:"mission_id"`
	Phase              string `json:"phase"`
	PhaseRunID         string `json:"phase_run_id,omitempty"`
	Role               string `json:"role"`
	InvocationID       string `json:"invocation_id,omitempty"`
	Attempt            int    `json:"attempt"`
	UsageScope         string `json:"usage_scope"`
	SessionID          string `json:"session_id,omitempty"`
	RecordKind         string `json:"record_kind"`
	TokensInput        *int64 `json:"tokens_input"`
	TokensOutput       *int64 `json:"tokens_output"`
	TokensCache        *int64 `json:"tokens_cache,omitempty"`
	TokensReasoning    *int64 `json:"tokens_reasoning,omitempty"`
	Provider           string `json:"provider,omitempty"`
	Model              string `json:"model,omitempty"`
	MeasurementSource  string `json:"measurement_source"`
	MeasurementQuality string `json:"measurement_quality"`
	DedupUnavailable   bool   `json:"dedup_unavailable,omitempty"`
	ObservedAt         string `json:"observed_at"`
}

// MissionUsagePath returns the default ledger path under strategistRoot.
func MissionUsagePath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(MissionUsageRelPath))
}

// MissionUsageRecordID derives the idempotency key of a record. An event id
// wins; otherwise an invocation id with attempt and scope identifies the
// observation (a cumulative snapshot also carries its session and counts, since
// a counter legitimately reports several values); a record with neither has no
// stable identity and gets a content key that never matches another
// observation, so it is accepted but flagged dedup_unavailable by the caller.
func MissionUsageRecordID(rec MissionUsageRecord) string {
	switch {
	case rec.EventID != "":
		return "event|" + rec.MissionID + "|" + rec.EventID
	case rec.InvocationID != "":
		id := fmt.Sprintf("inv|%s|%s|%d|%s", rec.MissionID, rec.InvocationID, rec.Attempt, rec.UsageScope)
		if rec.RecordKind == string(domain.UsageKindCumulativeSnapshot) {
			id += "|snap|" + rec.SessionID + "|" + usageCount(rec.TokensInput) + "|" + usageCount(rec.TokensOutput)
		}
		return id
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|%s",
		rec.MissionID, rec.Phase, rec.Role, rec.UsageScope, rec.RecordKind, rec.Attempt,
		usageCount(rec.TokensInput), usageCount(rec.TokensOutput), usageCount(rec.TokensCache), usageCount(rec.TokensReasoning), rec.ObservedAt)))
	return "raw|" + rec.MissionID + "|" + hex.EncodeToString(sum[:8])
}

func usageCount(p *int64) string {
	if p == nil {
		return "-"
	}
	return strconv.FormatInt(*p, 10)
}

// ValidateMissionUsageRecord checks a record before it is written.
func ValidateMissionUsageRecord(rec MissionUsageRecord) error {
	return errors.Join(usageIdentityErrors(rec), usageTokenErrors(rec), usageQualityErrors(rec))
}

func usageIdentityErrors(rec MissionUsageRecord) error {
	var errs []error
	if rec.SchemaVersion != MissionUsageSchemaVersion {
		errs = append(errs, fmt.Errorf("schema_version must be %q, got %q", MissionUsageSchemaVersion, rec.SchemaVersion))
	}
	for name, v := range map[string]string{"record_id": rec.RecordID, "mission_id": rec.MissionID, "role": rec.Role, "usage_scope": rec.UsageScope} {
		if v == "" {
			errs = append(errs, fmt.Errorf("%s is required", name))
		}
	}
	if !domain.ValidUsagePhase(rec.Phase) {
		errs = append(errs, fmt.Errorf("phase %q is not a usage phase (the Approval Gate carries no tokens)", rec.Phase))
	}
	if rec.Attempt < 1 {
		errs = append(errs, fmt.Errorf("attempt must be >= 1, got %d", rec.Attempt))
	}
	if _, err := time.Parse(time.RFC3339, rec.ObservedAt); err != nil {
		errs = append(errs, fmt.Errorf("observed_at must be RFC 3339: %w", err))
	}
	return errors.Join(errs...)
}

func usageTokenErrors(rec MissionUsageRecord) error {
	var errs []error
	if !domain.ValidUsageRecordKind(rec.RecordKind) {
		errs = append(errs, fmt.Errorf("record_kind %q is not a usage record kind", rec.RecordKind))
	}
	if rec.RecordKind == string(domain.UsageKindCumulativeSnapshot) && rec.SessionID == "" {
		errs = append(errs, errors.New("a cumulative_snapshot requires session_id: deltas are only valid inside one session"))
	}
	if rec.TokensInput == nil && rec.TokensOutput == nil {
		errs = append(errs, errors.New("at least one of tokens_input or tokens_output must be observed"))
	}
	return errors.Join(append(errs, negativeCountErrors(rec))...)
}

func negativeCountErrors(rec MissionUsageRecord) error {
	var errs []error
	counts := map[string]*int64{"tokens_input": rec.TokensInput, "tokens_output": rec.TokensOutput, "tokens_cache": rec.TokensCache, "tokens_reasoning": rec.TokensReasoning}
	for name, p := range counts {
		if p != nil && *p < 0 {
			errs = append(errs, fmt.Errorf("%s must be >= 0, got %d", name, *p))
		}
	}
	return errors.Join(errs...)
}

func usageQualityErrors(rec MissionUsageRecord) error {
	source, quality := domain.UsageSource(rec.MeasurementSource), domain.UsageQuality(rec.MeasurementQuality)
	if !domain.UsageQualityAllowedForSource(source, quality) {
		return fmt.Errorf("measurement_quality %q is not allowed for measurement_source %q", rec.MeasurementQuality, rec.MeasurementSource)
	}
	return nil
}
