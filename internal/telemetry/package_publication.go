package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RefinedPackagePublicationHistoryRelPath is the runtime-relative publication ledger path.
const RefinedPackagePublicationHistoryRelPath = "memory/refined-package-publications.jsonl"

// RefinedPackagePublicationRecord identifies one canonical package publication.
// It is intentionally separate from SniperMaterializationRecord: publishing a
// package is not evidence that a documentation target was written.
type RefinedPackagePublicationRecord struct {
	MissionID        string    `json:"mission_id"`
	ProviderChangeID string    `json:"provider_change_id"`
	SourceDigest     string    `json:"source_digest"`
	PackageDigest    string    `json:"package_digest"`
	PublishedAt      time.Time `json:"published_at"`
}

// RefinedPackagePublicationHistoryPath returns the publication ledger path.
func RefinedPackagePublicationHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(RefinedPackagePublicationHistoryRelPath))
}

// AppendRefinedPackagePublication appends one publication event. It does not
// share the materialization ledger, so two package publications and one target
// write remain distinguishable in incident reconstruction.
func AppendRefinedPackagePublication(path string, record RefinedPackagePublicationRecord) error {
	if record.MissionID == "" || record.ProviderChangeID == "" || record.SourceDigest == "" || record.PackageDigest == "" || record.PublishedAt.IsZero() {
		return fmt.Errorf("refined package publication: mission, change, source digest, package digest and timestamp are required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create refined package publication history: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // history path is runtime-owned
	if err != nil {
		return fmt.Errorf("open refined package publication history: %w", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // close errors cannot change an already appended event
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal refined package publication: %w", err)
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("append refined package publication: %w", err)
	}
	return nil
}

// ReadRefinedPackagePublications reads valid publication records and skips
// malformed historical lines, matching the other runtime JSONL histories.
func ReadRefinedPackagePublications(path string) ([]RefinedPackagePublicationRecord, error) {
	f, err := os.Open(path) //nolint:gosec // history path is runtime-owned
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open refined package publication history: %w", err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read result is already determined
	var records []RefinedPackagePublicationRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record RefinedPackagePublicationRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read refined package publication history: %w", err)
	}
	return records, nil
}
