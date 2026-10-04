package main

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
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

func missionPath(root, id string) string { return missionruntime.Path(root, id) }

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

func writeMissionResult(cmd *cobra.Command, asJSON bool, value any) error {
	if asJSON {
		return encodeMissionResult(cmd, value)
	}
	status, ok := value.(domain.MissionEngineStatus)
	if !ok {
		return encodeMissionResult(cmd, value)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s phase=%s state=%s\n", status.MissionID, status.Phase, status.State); err != nil {
		return fmt.Errorf("write mission result: %w", err)
	}
	return nil
}

func encodeMissionResult(cmd *cobra.Command, value any) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(value); err != nil {
		return fmt.Errorf("encode mission result: %w", err)
	}
	return nil
}
