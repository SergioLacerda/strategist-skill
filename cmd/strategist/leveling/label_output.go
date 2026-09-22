package leveling

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

// WriteLabelResult renders the label result to stdout/stderr.
func WriteLabelResult(cmd *cobra.Command, reg domain.RoleRegistry, result LabelResult, opts LabelOptions) error {
	if err := WriteLabelWarning(cmd, result.Warning); err != nil {
		return err
	}
	rendered := internal.RenderWith(reg, result.Level, opts.Message, opts.Width)
	if opts.JSON {
		return WriteJSONLabel(cmd, result, rendered)
	}
	return WritePlainLabel(cmd, result.Level.Label(), rendered, opts.Message != "")
}

// WriteLabelWarning writes a non-fatal label warning.
func WriteLabelWarning(cmd *cobra.Command, warning string) error {
	if warning == "" {
		return nil
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}

// WriteJSONLabel writes the JSON payload for `leveling label --json`.
func WriteJSONLabel(cmd *cobra.Command, result LabelResult, rendered string) error {
	payload := struct {
		internal.Level
		Label    string `json:"label"`
		Tag      string `json:"tag"`
		Rendered string `json:"rendered"`
		Reused   bool   `json:"reused"`
	}{result.Level, result.Level.Label(), result.Level.Tag(), rendered, result.Reused}
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(payload); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}

// WritePlainLabel writes the plain text label output.
func WritePlainLabel(cmd *cobra.Command, label, rendered string, useRendered bool) error {
	if useRendered {
		label = rendered
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), label); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}
