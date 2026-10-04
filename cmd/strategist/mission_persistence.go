package main

import "fmt"

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
)

func wrapMissionError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w", err)
}

// These small adapters preserve the existing package-main seams while the
// mission package owns persistence policy and filesystem behavior.
func validateMissionID(id string) error {
	return wrapMissionError(missionruntime.ValidateMissionID(id))
}

func requireMissionID(id string) error {
	if err := missionruntime.ValidateMissionID(id); err != nil {
		return fmt.Errorf("mission: %w", err)
	}
	return nil
}

func saveMission(root string, status domain.MissionEngineStatus) error {
	return wrapMissionError(missionruntime.Save(root, status))
}

func lockMission(root, missionID string, fn func() error) error {
	return wrapMissionError(missionruntime.Lock(root, missionID, fn))
}

func loadMission(root, id string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	engine, status, err := missionruntime.Load(root, id)
	return engine, status, wrapMissionError(err)
}
