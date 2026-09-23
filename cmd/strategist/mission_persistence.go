package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// missionIDPattern restricts --mission-id to the same safe character set
// GenerateMissionID (internal/install/mission_id.go) produces:
// lowercase/digits/hyphens. This also protects filesystem lookups keyed by
// the id (missionPath, missionIDKnown's filepath.Glob) from path-traversal or
// glob-metacharacter injection via an operator-supplied string.
var missionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// validateMissionID is the single --mission-id check shared by every mission
// subcommand. It returns an unprefixed error; callers add their command prefix.
func validateMissionID(id string) error {
	if id == "" {
		return errors.New("--mission-id is required")
	}
	if !missionIDPattern.MatchString(id) {
		return fmt.Errorf("--mission-id %q is malformed (want lowercase letters, digits, and hyphens, e.g. 20260830-skill-gaps-triage)", id)
	}
	return nil
}

func requireMissionID(id string) error {
	if err := validateMissionID(id); err != nil {
		return fmt.Errorf("mission: %w", err)
	}
	return nil
}

func missionPath(root, id string) string { return filepath.Join(root, "missions", id+".json") }

func saveMission(root string, status domain.MissionEngineStatus) error {
	if err := os.MkdirAll(filepath.Join(root, "missions"), 0o755); err != nil {
		return fmt.Errorf("create mission directory: %w", err)
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mission state: %w", err)
	}
	if err := os.WriteFile(missionPath(root, status.MissionID), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write mission state: %w", err)
	}
	return nil
}

func loadMission(root, id string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	data, err := os.ReadFile(missionPath(root, id))
	if err != nil {
		return nil, domain.MissionEngineStatus{}, fmt.Errorf("mission %q not found: %w", id, err)
	}
	var status domain.MissionEngineStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, status, fmt.Errorf("invalid persisted state: %w", err)
	}
	engine, err := domain.RestoreMission(status)
	if err != nil {
		return nil, status, fmt.Errorf("restore mission state: %w", err)
	}
	return engine, status, nil
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
