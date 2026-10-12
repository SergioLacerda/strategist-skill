package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ResolveOpenPhaseRun returns the only currently open run for a phase. It is
// intentionally read-only: usage attribution must use a durable lifecycle
// marker, never reconstruct a run id from a regenerated history or from the
// phase name alone.
func ResolveOpenPhaseRun(path, missionID, phase string) (string, error) {
	records, err := ReadMissionLifecycleFor(path, missionID)
	if err != nil {
		return "", fmt.Errorf("read lifecycle for phase-run attribution: %w", err)
	}
	if len(records) == 0 {
		// A legacy mission may predate the lifecycle ledger. Preserve its
		// readable usage record; there is no historical authority from which
		// to infer a run.
		return "", nil
	}
	open, err := openPhaseRun(records, phase)
	if err != nil {
		return "", err
	}
	if open == "" {
		return "", fmt.Errorf("phase %q has no open lifecycle run", phase)
	}
	return open, nil
}

func openPhaseRun(records []MissionLifecycleRecord, phase string) (string, error) {
	open := ""
	for _, record := range records {
		if record.Phase != phase {
			continue
		}
		var err error
		open, err = foldOpenPhaseRun(open, phase, record)
		if err != nil {
			return "", err
		}
	}
	return open, nil
}

func foldOpenPhaseRun(open, phase string, record MissionLifecycleRecord) (string, error) {
	switch record.MarkerKind {
	case string(domain.MarkerPhaseStarted):
		if open != "" && open != record.PhaseRunID {
			return "", fmt.Errorf("phase %q has ambiguous open runs %q and %q", phase, open, record.PhaseRunID)
		}
		return record.PhaseRunID, nil
	case string(domain.MarkerPhaseEnded):
		if record.PhaseRunID == open {
			return "", nil
		}
	}
	return open, nil
}

// AppendMissionLifecycleRecords appends recs to the ledger under an exclusive
// lock and skips every record whose marker_id already exists, so a replayed
// marker is a no-op. The whole batch is validated first: one invalid record
// rejects the batch and nothing is written. It reports how many lines were
// appended. The file and its directory are created when absent.
func AppendMissionLifecycleRecords(path string, recs []MissionLifecycleRecord) (appended int, err error) {
	written, err := AppendNewMissionLifecycleRecords(path, recs)
	return len(written), err
}

// AppendNewMissionLifecycleRecords is AppendMissionLifecycleRecords returning
// the records that were actually written (not the duplicates), so an exporter
// can emit exactly one event per new marker.
func AppendNewMissionLifecycleRecords(path string, recs []MissionLifecycleRecord) (written []MissionLifecycleRecord, err error) {
	if err = validateLifecycleBatch(recs); err != nil {
		return nil, err
	}
	err = withLockedJSONL(path, "mission lifecycle ledger", func(f *os.File) error {
		var appendErr error
		written, appendErr = appendNewLifecycleRecords(f, recs)
		return appendErr
	})
	return written, err
}

func validateLifecycleBatch(recs []MissionLifecycleRecord) error {
	for _, rec := range recs {
		if err := ValidateMissionLifecycleRecord(rec); err != nil {
			return fmt.Errorf("mission lifecycle validation failed: %w", err)
		}
	}
	return nil
}

func appendNewLifecycleRecords(f *os.File, recs []MissionLifecycleRecord) ([]MissionLifecycleRecord, error) {
	seen, err := lifecycleMarkerIDs(f)
	if err != nil {
		return nil, err
	}
	var written []MissionLifecycleRecord
	for _, rec := range recs {
		if seen[rec.MarkerID] {
			continue
		}
		if err := writeLifecycleRecord(f, rec); err != nil {
			return written, err
		}
		seen[rec.MarkerID] = true
		written = append(written, rec)
	}
	return written, nil
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
