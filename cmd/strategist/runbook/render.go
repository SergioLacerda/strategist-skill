package runbook

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
)

func render(cmd *cobra.Command, projection jeweler.Projection, format string) error {
	rows, err := selectionRows(projection.Items)
	if err != nil {
		return err
	}
	if format == "json" {
		return renderJSON(cmd, rows)
	}
	if format != "" && format != "table" {
		return fmt.Errorf("runbook select: unknown --format %q (want table or json)", format)
	}
	return renderTable(cmd, rows)
}

func selectionRows(items []jeweler.Item) ([]selectionRow, error) {
	rows := make([]selectionRow, 0, len(items))
	for i, item := range items {
		if item.Source.Path == "" {
			return nil, fmt.Errorf("runbook select: JEWELER item %q has no source path", item.ID)
		}
		role := "supporting"
		if i == 0 {
			role = "primary"
		}
		rows = append(rows, selectionRow{RunbookID: runbookID(item), Role: role, ChestID: "runbooks", Ref: item.Source.Path, Reason: item.Reason})
	}
	return rows, nil
}

func renderJSON(cmd *cobra.Command, rows []selectionRow) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(rows); err != nil {
		return fmt.Errorf("render runbook selection JSON: %w", err)
	}
	return nil
}

func renderTable(cmd *cobra.Command, rows []selectionRow) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "[Strategist] runbook select: no runbook matched the given signals")
		if err != nil {
			return fmt.Errorf("render empty runbook selection: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "RUNBOOK_ID\tROLE\tREF\tREASON"); err != nil {
		return fmt.Errorf("render runbook selection header: %w", err)
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", row.RunbookID, row.Role, row.Ref, row.Reason); err != nil {
			return fmt.Errorf("render runbook selection row: %w", err)
		}
	}
	return nil
}

func runbookID(item jeweler.Item) string {
	if item.Source.Path == "" {
		return item.ID
	}
	base := filepath.Base(item.Source.Path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
