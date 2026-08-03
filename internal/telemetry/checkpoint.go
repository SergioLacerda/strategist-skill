package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MissionCheckpoint tracks which tasks the Sniper has completed.
// Written atomically after each task; read at Sniper start to skip already-done tasks.
type MissionCheckpoint struct {
	MissionID      string    `json:"mission_id"`
	TasksTotal     int       `json:"tasks_total"`
	TasksCompleted []int     `json:"tasks_completed"`
	LastUpdated    time.Time `json:"last_updated"`
}

// TaskDone reports whether task number n is already in the completed list.
func (c *MissionCheckpoint) TaskDone(n int) bool {
	for _, done := range c.TasksCompleted {
		if done == n {
			return true
		}
	}
	return false
}

// MarkTaskDone appends n to TasksCompleted if not already present.
func (c *MissionCheckpoint) MarkTaskDone(n int) {
	if !c.TaskDone(n) {
		c.TasksCompleted = append(c.TasksCompleted, n)
	}
	c.LastUpdated = time.Now()
}

// LoadCheckpoint reads a checkpoint file. Returns an empty checkpoint (not an error)
// if the file does not exist — a missing file means no tasks have been completed yet.
func LoadCheckpoint(path string) (*MissionCheckpoint, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: checkpoint path is owned by the Strategist runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return &MissionCheckpoint{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	var cp MissionCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint: %w", err)
	}
	return &cp, nil
}

// SaveCheckpoint writes the checkpoint atomically: temp file + rename.
// Concurrent writers on the same path are safe because rename is atomic on POSIX.
func SaveCheckpoint(path string, cp *MissionCheckpoint) error {
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp checkpoint: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()        //nolint:errcheck,gosec // best-effort cleanup on write failure
		os.Remove(tmpName) //nolint:errcheck,gosec // best-effort cleanup on write failure
		return fmt.Errorf("write temp checkpoint: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName) //nolint:errcheck,gosec // best-effort cleanup on close failure
		return fmt.Errorf("close temp checkpoint: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName) //nolint:errcheck,gosec // best-effort cleanup on rename failure
		return fmt.Errorf("rename checkpoint: %w", err)
	}
	return nil
}

// RemoveCheckpoint deletes the checkpoint file after a mission completes successfully.
// A missing file is not an error.
func RemoveCheckpoint(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove checkpoint: %w", err)
	}
	return nil
}
