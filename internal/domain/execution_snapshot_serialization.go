package domain

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ArtifactPaths returns a stable path projection for legacy telemetry callers.
func (s ExecutionSnapshot) ArtifactPaths() []string {
	paths := make([]string, 0)
	for _, group := range s.Groups {
		for _, artifact := range group.Artifacts {
			paths = append(paths, artifact.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

// MarshalJSON ensures a zero-value metadata map never changes the durable
// shape and keeps validation at the boundary where snapshots are persisted.
func (s ExecutionSnapshot) MarshalJSON() ([]byte, error) {
	type snapshotAlias ExecutionSnapshot
	raw, err := json.Marshal(snapshotAlias(s))
	if err != nil {
		return nil, fmt.Errorf("marshal execution snapshot: %w", err)
	}
	return raw, nil
}
