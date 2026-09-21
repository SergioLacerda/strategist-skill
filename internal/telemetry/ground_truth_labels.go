package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GroundTruthLabelHistoryRelPath is relative to the .strategist runtime root.
// One label file serves every subject whose correctness is decided after the
// fact (Scout routes, downstream Handoff Challenge application), so metrics
// never infer ground truth from decisions alone.
const GroundTruthLabelHistoryRelPath = "memory/ground-truth-labels.jsonl"

// GroundTruthLabel is one reviewed outcome. Ref and Kind are mandatory: a
// label without a declared source is not ground truth.
type GroundTruthLabel struct {
	MissionID string `json:"mission_id"`
	Subject   string `json:"subject"`
	Label     string `json:"label"`
	Kind      string `json:"ground_truth_kind"`
	Ref       string `json:"ground_truth_ref"`
	Timestamp string `json:"timestamp"`
}

// GroundTruthLabelHistoryPath returns the default label history path.
func GroundTruthLabelHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(GroundTruthLabelHistoryRelPath))
}

// ValidateGroundTruthLabel enforces subject/label vocabulary and provenance.
func ValidateGroundTruthLabel(l GroundTruthLabel) error {
	var errs []error
	errs = append(errs, requiredRouteField("mission_id", l.MissionID)...)
	errs = append(errs, requiredRouteField("ground_truth_ref", l.Ref)...)
	errs = append(errs, validateLabelProvenance(l)...)
	errs = append(errs, validateLabelVocabulary(l)...)
	errs = append(errs, validateLabelTimestamp(l.Timestamp)...)
	return errors.Join(errs...)
}

func validateLabelProvenance(l GroundTruthLabel) []error {
	var errs []error
	if !groundTruthKinds[l.Kind] {
		errs = append(errs, fmt.Errorf("ground_truth_kind %q is not an allowed value", l.Kind))
	}
	if l.Subject == GroundTruthSubjectGateOutcome && !strings.HasPrefix(l.Ref, "approval_gate:") {
		errs = append(errs, errors.New("gate_outcome labels require an approval_gate provenance reference"))
	}
	return errs
}

func validateLabelVocabulary(l GroundTruthLabel) []error {
	labels, ok := groundTruthLabelsBySubject[l.Subject]
	if !ok {
		return []error{fmt.Errorf("subject %q is not an allowed value", l.Subject)}
	}
	if !labels[l.Label] {
		return []error{fmt.Errorf("label %q is not allowed for subject %q", l.Label, l.Subject)}
	}
	return nil
}

func validateLabelTimestamp(ts string) []error {
	if ts == "" {
		return requiredRouteField("timestamp", ts)
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		return []error{fmt.Errorf("timestamp %q is not RFC3339", ts)}
	}
	return nil
}

// AppendGroundTruthLabel validates and appends a label unless one already
// exists for the same mission_id and subject (first review wins, so replays
// cannot inflate the sample). appended reports whether a line was written.
func AppendGroundTruthLabel(path string, label GroundTruthLabel) (appended bool, err error) {
	if err = ValidateGroundTruthLabel(label); err != nil {
		return false, fmt.Errorf("ground truth label validation failed: %w", err)
	}
	f, err := openGroundTruthLabelFile(path)
	if err != nil {
		return false, err
	}
	defer closeFileWithContext(f, &err, "close ground truth labels")
	if err = lockFileExclusive(f); err != nil {
		return false, fmt.Errorf("lock ground truth labels: %w", err)
	}
	defer func() {
		if unlockErr := unlockFile(f); unlockErr != nil && err == nil {
			err = fmt.Errorf("unlock ground truth labels: %w", unlockErr)
		}
	}()
	return appendGroundTruthLabelLocked(f, label)
}

func openGroundTruthLabelFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("ground truth label: create parent: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // path is owned by runtime memory
	if err != nil {
		return nil, fmt.Errorf("open ground truth labels: %w", err)
	}
	return f, nil
}

func appendGroundTruthLabelLocked(f *os.File, label GroundTruthLabel) (bool, error) {
	existing, err := scanGroundTruthLabels(f)
	if err != nil {
		return false, err
	}
	if hasGroundTruthLabel(existing, label) {
		return false, nil
	}
	encoded, err := json.Marshal(label)
	if err != nil {
		return false, fmt.Errorf("encode ground truth label: %w", err)
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return false, fmt.Errorf("seek ground truth labels: %w", err)
	}
	if _, err := fmt.Fprintln(f, string(encoded)); err != nil {
		return false, fmt.Errorf("write ground truth label: %w", err)
	}
	return true, nil
}

func hasGroundTruthLabel(existing []GroundTruthLabel, label GroundTruthLabel) bool {
	for _, e := range existing {
		if e.MissionID == label.MissionID && e.Subject == label.Subject {
			return true
		}
	}
	return false
}

// ReadGroundTruthLabels reads valid labels for subject ("" reads all). A
// missing file yields a nil slice; malformed or invalid lines are skipped, and
// the first label per mission_id and subject wins.
func ReadGroundTruthLabels(path, subject string) (_ []GroundTruthLabel, err error) {
	f, err := os.Open(path) //nolint:gosec // path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open ground truth labels: %w", err)
	}
	defer closeFileWithContext(f, &err, "close ground truth labels")
	all, err := scanGroundTruthLabels(f)
	if err != nil {
		return nil, err
	}
	return filterLabelsBySubject(all, subject), nil
}

func filterLabelsBySubject(all []GroundTruthLabel, subject string) (labels []GroundTruthLabel) {
	for _, l := range all {
		if subject == "" || l.Subject == subject {
			labels = append(labels, l)
		}
	}
	return labels
}

func scanGroundTruthLabels(r io.Reader) ([]GroundTruthLabel, error) {
	var labels []GroundTruthLabel
	seen := map[string]bool{}
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		var l GroundTruthLabel
		if json.Unmarshal(scanner.Bytes(), &l) != nil || ValidateGroundTruthLabel(l) != nil {
			continue
		}
		key := l.Subject + "\x00" + l.MissionID
		if seen[key] {
			continue
		}
		seen[key] = true
		labels = append(labels, l)
	}
	return labels, jsonlScannerErr(scanner, "scan ground truth labels")
}
