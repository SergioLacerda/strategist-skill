package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// List summarizes the request records of one mission, or of every mission when
// missionID is empty, ordered by creation time. It is read-only: it never
// changes a record, never takes a lease and never exposes a payload, input or
// nonce. A pending request past its expiry is flagged as expired, not hidden;
// a record that cannot be read is reported by file name only.
func (s InvocationStore) List(missionID string) (domain.MissionInvocationListing, error) {
	listing := domain.MissionInvocationListing{Requests: []domain.MissionInvocationSummary{}}
	dir, entries, err := s.recordEntries()
	if err != nil {
		return listing, err
	}
	for _, entry := range entries {
		s.collect(&listing, filepath.Join(dir, entry.Name()), missionID)
	}
	sortSummaries(listing.Requests)
	return listing, nil
}

// recordEntries returns the record files of the store; a store that was never
// written to has none, which is not an error.
func (s InvocationStore) recordEntries() (string, []os.DirEntry, error) {
	if s.Root == "" {
		return "", nil, fmt.Errorf("mission invocation: root is required")
	}
	dir := filepath.Join(s.Root, "missions", "invocations")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return dir, nil, nil
	}
	if err != nil {
		return dir, nil, fmt.Errorf("mission invocation: list requests: %w", err)
	}
	records := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			records = append(records, entry)
		}
	}
	return dir, records, nil
}

func (s InvocationStore) collect(listing *domain.MissionInvocationListing, path, missionID string) {
	summary, ok := s.summarize(path)
	switch {
	case !ok:
		listing.Skipped = append(listing.Skipped, filepath.Base(path))
	case missionID == "" || summary.MissionID == missionID:
		listing.Requests = append(listing.Requests, summary)
	}
}

func sortSummaries(summaries []domain.MissionInvocationSummary) {
	sort.SliceStable(summaries, func(i, j int) bool {
		a, b := summaries[i], summaries[j]
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.RequestID < b.RequestID
	})
}

func (s InvocationStore) summarize(path string) (domain.MissionInvocationSummary, bool) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is listed from the store directory.
	if err != nil {
		return domain.MissionInvocationSummary{}, false
	}
	var record domain.MissionInvocationRecord
	if json.Unmarshal(raw, &record) != nil || record.Request.RequestID == "" {
		return domain.MissionInvocationSummary{}, false
	}
	state := record.EffectiveState()
	return domain.MissionInvocationSummary{
		RequestID: record.Request.RequestID, MissionID: record.Request.MissionID,
		Role: record.Request.Role, Slot: record.Request.Slot, State: state,
		CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt,
		Expired:          state == domain.InvocationStatePending && !record.ExpiresAt.IsZero() && s.now().After(record.ExpiresAt),
		ExecutionAdapter: record.EffectiveAdapter(),
	}, true
}
