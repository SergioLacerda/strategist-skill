package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrMissionUsageConflict reports a record whose idempotency key already exists
// with different counts: it is neither a duplicate nor a new observation.
var ErrMissionUsageConflict = errors.New("mission usage record conflicts with an existing record of the same key")

// AppendMissionUsageRecord validates rec and appends it under an exclusive
// lock. It reports false when the same key and the same counts already exist
// (a duplicate delivery) and ErrMissionUsageConflict when the key exists with
// different counts. The file and its directory are created when absent.
func AppendMissionUsageRecord(path string, rec MissionUsageRecord) (appended bool, err error) {
	if err = ValidateMissionUsageRecord(rec); err != nil {
		return false, fmt.Errorf("mission usage validation failed: %w", err)
	}
	err = withLockedJSONL(path, "mission usage ledger", func(f *os.File) error {
		var appendErr error
		appended, appendErr = appendUsageUnlessKnown(f, rec)
		return appendErr
	})
	return appended, err
}

func appendUsageUnlessKnown(f *os.File, rec MissionUsageRecord) (bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek mission usage ledger: %w", err)
	}
	existing, err := scanMissionUsage(f)
	if err != nil {
		return false, err
	}
	if prior, found := findUsageRecord(existing, rec.RecordID); found {
		if sameUsageCounts(prior, rec) {
			return false, nil
		}
		return false, fmt.Errorf("%w: %s", ErrMissionUsageConflict, rec.RecordID)
	}
	return true, writeUsageRecord(f, rec)
}

func findUsageRecord(recs []MissionUsageRecord, id string) (MissionUsageRecord, bool) {
	for _, r := range recs {
		if r.RecordID == id {
			return r, true
		}
	}
	return MissionUsageRecord{}, false
}

func writeUsageRecord(f *os.File, rec MissionUsageRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal mission usage record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append mission usage record: %w", err)
	}
	return nil
}

func sameUsageCounts(a, b MissionUsageRecord) bool {
	return usageCount(a.TokensInput) == usageCount(b.TokensInput) &&
		usageCount(a.TokensOutput) == usageCount(b.TokensOutput) &&
		usageCount(a.TokensCache) == usageCount(b.TokensCache) &&
		usageCount(a.TokensReasoning) == usageCount(b.TokensReasoning)
}

// ReadMissionUsageFor reads the ledger and keeps one mission's records in file
// order. A missing file is an empty history; malformed lines are skipped.
func ReadMissionUsageFor(path, missionID string) (recs []MissionUsageRecord, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: ledger path is owned by the runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open mission usage ledger: %w", err)
	}
	defer closeFileWithContext(f, &err, "close mission usage ledger")
	all, err := scanMissionUsage(f)
	if err != nil {
		return nil, err
	}
	for _, rec := range all {
		if rec.MissionID == missionID {
			recs = append(recs, rec)
		}
	}
	return recs, nil
}

func scanMissionUsage(r io.Reader) ([]MissionUsageRecord, error) {
	var recs []MissionUsageRecord
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		var rec MissionUsageRecord
		if len(scanner.Bytes()) == 0 || json.Unmarshal(scanner.Bytes(), &rec) != nil {
			continue
		}
		recs = append(recs, rec)
	}
	return recs, jsonlScannerErr(scanner, "scan mission usage ledger")
}
