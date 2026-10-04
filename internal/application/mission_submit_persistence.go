package application

import (
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionSubmitPersistencePorts are the state and analysis rollback seams
// used while committing one submit transition.
type MissionSubmitPersistencePorts struct {
	Save            func(root string, status domain.MissionEngineStatus) error
	RestoreAnalysis func(path string, original []byte) error
}

// PersistMissionSubmitState saves the FSM state and restores analysis.md when
// the prior gate-materialization step changed it but the state save fails.
// Filesystem and durable storage remain injected at the composition root.
func PersistMissionSubmitState(root string, status domain.MissionEngineStatus, analysisPath string, original []byte, changed bool, ports MissionSubmitPersistencePorts) error {
	if ports.Save == nil {
		return errors.New("mission submit: state persistence port is required")
	}
	if err := ports.Save(root, status); err != nil {
		if restoreErr := rollbackSubmitAnalysis(analysisPath, original, changed, ports.RestoreAnalysis); restoreErr != nil {
			return fmt.Errorf("mission submit: %v; rollback failed: %w", err, restoreErr)
		}
		return fmt.Errorf("mission submit: %w", err)
	}
	return nil
}

func rollbackSubmitAnalysis(path string, original []byte, changed bool, restore func(string, []byte) error) error {
	if !changed || restore == nil {
		return nil
	}
	return restore(path, original)
}
