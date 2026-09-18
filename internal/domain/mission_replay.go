package domain

import "fmt"

// MissionReplayEvent is ordered evidence submitted through MissionEngine.
type MissionReplayEvent struct {
	Sequence uint64
	Event    MissionEngineEvent
}

// ReplayMission restores a snapshot and applies contiguous evidence through
// the same transition authority used by live events.
func ReplayMission(snapshot MissionEngineStatus, events []MissionReplayEvent) (*MissionEngine, error) {
	engine, err := RestoreMission(snapshot)
	if err != nil {
		return nil, fmt.Errorf("mission replay: %w", err)
	}
	for i, item := range events {
		want := uint64(i + 1)
		if item.Sequence != want {
			return nil, fmt.Errorf("mission replay: sequence gap at %d (got %d)", want, item.Sequence)
		}
		if _, err := engine.Submit(item.Event); err != nil {
			return nil, fmt.Errorf("mission replay: event %d: %w", item.Sequence, err)
		}
	}
	return engine, nil
}
