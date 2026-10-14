package mission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// ReadCompletion reads exactly one host completion object from stdin. The
// transport parser belongs to the Cobra adapter; semantic completion checks
// remain owned by internal/application.
func ReadCompletion(cmd *cobra.Command) (domain.MissionInvocationCompletion, error) {
	input, err := cliutil.ReadInput(cmd.InOrStdin())
	if err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var raw json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(input))
	if err := decoder.Decode(&raw); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: more than one object was supplied")
	} else if err != io.EOF {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: trailing data: %w", err)
	}
	if err := application.ValidateCompletionObject(raw); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var completion domain.MissionInvocationCompletion
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&completion); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	return completion, nil
}
