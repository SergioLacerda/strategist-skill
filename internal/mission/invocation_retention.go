package mission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// invocationRetention is how long consumed or expired records stay on disk.
const invocationRetention = 24 * time.Hour

// prune removes records that were consumed or expired more than
// invocationRetention ago. It is best-effort: a failure never blocks a request.
func (s InvocationStore) prune(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := s.now().Add(-invocationRetention)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if s.recordIsStale(path, cutoff) {
			_ = os.Remove(path) //nolint:errcheck // best-effort retention cleanup.
		}
	}
}

func (s InvocationStore) recordIsStale(path string, cutoff time.Time) bool {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is listed from the store directory.
	if err != nil {
		return false
	}
	var record domain.MissionInvocationRecord
	if json.Unmarshal(raw, &record) != nil {
		return false
	}
	if record.EffectiveState() == domain.InvocationStateCompleted {
		finished := record.CompletedAt
		if finished == nil {
			finished = record.ConsumedAt
		}
		return finished != nil && finished.Before(cutoff)
	}
	return !record.ExpiresAt.IsZero() && record.ExpiresAt.Before(cutoff)
}
