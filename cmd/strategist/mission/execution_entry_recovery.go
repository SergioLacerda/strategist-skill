package mission

import (
	"errors"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/application"
)

func executionRecoverySnapshot(entry executionEntry) application.ExecutionRecoverySnapshot {
	return application.ExecutionRecoverySnapshot{
		ID: entry.ID, MissionID: entry.MissionID, Targets: append([]string(nil), entry.Targets...),
		Outcome: entry.Outcome, DesiredStatus: entry.DesiredStatus, StateSaved: entry.StateSaved, Completed: entry.Completed,
	}
}

func loadRecoverableExecutionEntry(root, missionID string) (executionEntry, bool, error) {
	e, err := loadExecutionEntry(executionEntryPath(root, missionID))
	if errors.Is(err, os.ErrNotExist) || e.Completed {
		return e, false, nil
	}
	if err != nil {
		return e, false, fmt.Errorf("load execution entry: %w", err)
	}
	if e.MissionID != missionID || e.ID == "" {
		return e, false, fmt.Errorf("execution entry tampered: identity mismatch")
	}
	return e, true, nil
}
