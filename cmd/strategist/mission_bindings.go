package main

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/mission/bindingdrift"
)

// recordMissionBindings captures the slot bindings a mission starts with.
func recordMissionBindings(root, missionID string) error {
	slots, err := bindingdrift.CurrentSlots(root)
	if err != nil {
		return fmt.Errorf("record mission bindings: %w", err)
	}
	if err := bindingdrift.Save(root, missionID, slots, time.Now()); err != nil {
		return fmt.Errorf("record mission bindings: %w", err)
	}
	return nil
}

// missionBindingChanges lists the slots whose binding differs from the one the
// mission started with, as advisory lines. A mission without a record, or an
// unreadable configuration, yields none: the signal is advisory.
func missionBindingChanges(root, missionID string) []string {
	record, found, err := bindingdrift.Load(root, missionID)
	if err != nil || !found {
		return nil
	}
	current, err := bindingdrift.CurrentSlots(root)
	if err != nil {
		return nil
	}
	changes := bindingdrift.Diff(record.Slots, current)
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		lines = append(lines, change.String())
	}
	return lines
}
