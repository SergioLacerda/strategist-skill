package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

// runbookSelectionRow is one Selection rendered for CLI output, shaped to
// match selected_runbooks_hint's schema
// (handoff-ranger-to-archivist.schema.yaml) so --format json output is
// copy-pastable into a handoff artifact without reshaping.
type runbookSelectionRow struct {
	RunbookID string `json:"runbook_id"`
	Role      string `json:"role"`
	ChestID   string `json:"chest_id"`
	Ref       string `json:"ref"`
	Reason    string `json:"reason"`
}

// sortRunbookSelectionRows orders primary before supporting, then by
// runbook id — deterministic output, same convention as
// sortItemRows (treasure_chest_items_render.go).
func sortRunbookSelectionRows(rows []runbookSelectionRow) {
	sort.Slice(rows, func(i, k int) bool {
		if rows[i].Role != rows[k].Role {
			return rows[i].Role == string(runbook.RolePrimary)
		}
		return rows[i].RunbookID < rows[k].RunbookID
	})
}

func renderRunbookSelectionTable(rows []runbookSelectionRow) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "RUNBOOK_ID", "ROLE", "REF", "REASON"); err != nil {
		return fmt.Errorf("runbook select: write header: %w", err)
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.RunbookID, r.Role, r.Ref, r.Reason); err != nil {
			return fmt.Errorf("runbook select: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("runbook select: flush: %w", err)
	}
	return nil
}

func renderRunbookSelectionJSON(rows []runbookSelectionRow) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		return fmt.Errorf("runbook select: encode json: %w", err)
	}
	return nil
}
