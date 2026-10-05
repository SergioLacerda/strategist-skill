package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// AppendMissionLifecycleRecords appends recs to the ledger under an exclusive
// lock and skips every record whose marker_id already exists, so a replayed
// marker is a no-op. The whole batch is validated first: one invalid record
// rejects the batch and nothing is written. It reports how many lines were
// appended. The file and its directory are created when absent.
func AppendMissionLifecycleRecords(path string, recs []MissionLifecycleRecord) (appended int, err error) {
	if err = validateLifecycleBatch(recs); err != nil {
		return 0, err
	}
	err = withLockedJSONL(path, "mission lifecycle ledger", func(f *os.File) error {
		n, appendErr := appendNewLifecycleRecords(f, recs)
		appended = n
		return appendErr
	})
	return appended, err
}

func validateLifecycleBatch(recs []MissionLifecycleRecord) error {
	for _, rec := range recs {
		if err := ValidateMissionLifecycleRecord(rec); err != nil {
			return fmt.Errorf("mission lifecycle validation failed: %w", err)
		}
	}
	return nil
}

func appendNewLifecycleRecords(f *os.File, recs []MissionLifecycleRecord) (int, error) {
	seen, err := lifecycleMarkerIDs(f)
	if err != nil {
		return 0, err
	}
	appended := 0
	for _, rec := range recs {
		if seen[rec.MarkerID] {
			continue
		}
		if err := writeLifecycleRecord(f, rec); err != nil {
			return appended, err
		}
		seen[rec.MarkerID] = true
		appended++
	}
	return appended, nil
}

func writeLifecycleRecord(f *os.File, rec MissionLifecycleRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal mission lifecycle record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append mission lifecycle record: %w", err)
	}
	return nil
}

func lifecycleMarkerIDs(f *os.File) (map[string]bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek mission lifecycle ledger: %w", err)
	}
	recs, err := scanMissionLifecycle(f)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(recs))
	for _, rec := range recs {
		seen[rec.MarkerID] = true
	}
	return seen, nil
}

// ReadMissionLifecycle reads every record of the ledger. A missing file is an
// empty history, not an error; malformed lines are skipped, consistent with
// the other append-only ledgers.
func ReadMissionLifecycle(path string) (recs []MissionLifecycleRecord, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: ledger path is owned by the runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open mission lifecycle ledger: %w", err)
	}
	defer closeFileWithContext(f, &err, "close mission lifecycle ledger")
	return scanMissionLifecycle(f)
}

// ReadMissionLifecycleFor reads the ledger and keeps the records of one
// mission, in file order.
func ReadMissionLifecycleFor(path, missionID string) ([]MissionLifecycleRecord, error) {
	all, err := ReadMissionLifecycle(path)
	if err != nil {
		return nil, err
	}
	var out []MissionLifecycleRecord
	for _, rec := range all {
		if rec.MissionID == missionID {
			out = append(out, rec)
		}
	}
	return out, nil
}

func scanMissionLifecycle(r io.Reader) ([]MissionLifecycleRecord, error) {
	var recs []MissionLifecycleRecord
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		var rec MissionLifecycleRecord
		if len(scanner.Bytes()) == 0 || json.Unmarshal(scanner.Bytes(), &rec) != nil {
			continue
		}
		recs = append(recs, rec)
	}
	return recs, jsonlScannerErr(scanner, "scan mission lifecycle ledger")
}
