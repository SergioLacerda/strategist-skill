// Package bindingdrift records the slot bindings a mission started with and
// reports when the effective bindings changed afterwards. It exists because a
// reinstall once switched the refinement Weapon under a running mission and
// nothing said so until an invocation was refused.
package bindingdrift

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const recordsRelDir = "missions/bindings"

var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Record is the slot bindings a mission started with. It carries provider ids
// only: no file content, secrets or paths.
type Record struct {
	MissionID  string            `json:"mission_id"`
	CapturedAt string            `json:"captured_at"`
	Slots      map[string]string `json:"slots"`
}

// Change is one slot whose binding differs from the one a mission started with.
type Change struct {
	Slot        string
	StartedWith string
	Now         string
}

// String is the one-line advisory for a changed slot.
func (c Change) String() string {
	return fmt.Sprintf("binding_changed slot=%s started_with=%s now=%s", c.Slot, orNone(c.StartedWith), orNone(c.Now))
}

func orNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

// CurrentSlots reads the slots block of active.yaml. A missing file yields an
// empty set, not an error.
func CurrentSlots(strategistRoot string) (map[string]string, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "active.yaml")) //nolint:gosec // fixed file under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg struct {
		Slots map[string]string `yaml:"slots"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse active.yaml: %w", err)
	}
	if cfg.Slots == nil {
		cfg.Slots = map[string]string{}
	}
	return cfg.Slots, nil
}

func recordPath(strategistRoot, missionID string) (string, error) {
	if !safeID.MatchString(missionID) {
		return "", fmt.Errorf("binding record: unsafe mission id %q", missionID)
	}
	return filepath.Join(strategistRoot, filepath.FromSlash(recordsRelDir), missionID+".json"), nil
}

// Save records the bindings a mission started with. It never overwrites: the
// record is the baseline the drift is measured against.
func Save(strategistRoot, missionID string, slots map[string]string, now time.Time) error {
	path, err := recordPath(strategistRoot, missionID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	data, err := json.Marshal(Record{MissionID: missionID, CapturedAt: now.UTC().Format(time.RFC3339), Slots: slots})
	if err != nil {
		return fmt.Errorf("marshal binding record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create binding record dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // runtime state file, not a secret
		return fmt.Errorf("write binding record: %w", err)
	}
	return nil
}

// Load reads one mission's record; found is false when it has none.
func Load(strategistRoot, missionID string) (Record, bool, error) {
	path, err := recordPath(strategistRoot, missionID)
	if err != nil {
		return Record{}, false, err
	}
	return readRecord(path)
}

func readRecord(path string) (Record, bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path built from a validated mission id under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("read binding record: %w", err)
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Record{}, false, fmt.Errorf("parse binding record: %w", err)
	}
	return rec, true, nil
}

// Latest returns the most recently captured record, which is the baseline
// strategist check compares the workspace against.
func Latest(strategistRoot string) (Record, bool, error) {
	matches, err := filepath.Glob(filepath.Join(strategistRoot, filepath.FromSlash(recordsRelDir), "*.json"))
	if err != nil {
		return Record{}, false, fmt.Errorf("list binding records: %w", err)
	}
	var latest Record
	found := false
	for _, path := range matches {
		rec, ok := readRecordQuiet(path)
		if ok && (!found || rec.CapturedAt > latest.CapturedAt) {
			latest, found = rec, true
		}
	}
	return latest, found, nil
}

// readRecordQuiet skips an unreadable or malformed record: one bad file must
// not hide the drift of the others.
func readRecordQuiet(path string) (Record, bool) {
	rec, ok, err := readRecord(path)
	return rec, ok && err == nil
}

// Diff lists the slots whose binding differs, sorted by slot name. A slot
// present on only one side counts as a change.
func Diff(start, current map[string]string) []Change {
	seen := make(map[string]struct{}, len(start)+len(current))
	var changes []Change
	for slot, was := range start {
		seen[slot] = struct{}{}
		if now := current[slot]; now != was {
			changes = append(changes, Change{Slot: slot, StartedWith: was, Now: now})
		}
	}
	for slot, now := range current {
		if _, ok := seen[slot]; ok {
			continue
		}
		changes = append(changes, Change{Slot: slot, Now: now})
	}
	sort.Slice(changes, func(i, k int) bool { return changes[i].Slot < changes[k].Slot })
	return changes
}
