package mission

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// WriteResult is the common CLI transport for mission commands. Human output
// remains the historical compact status line; all other values and JSON mode
// use the existing newline-delimited JSON envelope.
func WriteResult(cmd *cobra.Command, asJSON bool, value any) error {
	if asJSON {
		return encodeResult(cmd, value)
	}
	status, ok := value.(domain.MissionEngineStatus)
	if !ok {
		return encodeResult(cmd, value)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s phase=%s state=%s\n", status.MissionID, status.Phase, status.State); err != nil {
		return fmt.Errorf("write mission result: %w", err)
	}
	return nil
}

func encodeResult(cmd *cobra.Command, value any) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(value); err != nil {
		return fmt.Errorf("encode mission result: %w", err)
	}
	return nil
}
