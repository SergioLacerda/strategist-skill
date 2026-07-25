package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

// jewelJSONEntry is the shared JSON shape for jewel listings. Scope is
// omitted (zero value) by callers that don't expose applicability scope
// (e.g. `jewel list`); `mine list` includes it.
type jewelJSONEntry struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Status       string   `json:"status"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
	Scope        []string `json:"scope,omitempty"`
}

// renderJewelTable renders a jewel listing as a tab-aligned table. showStatus
// controls whether the STATUS column is included, matching the two current
// callers: `jewel list` (shows status) and `mine list` (proposed-only, no
// status column).
func renderJewelTable(jewels []treasure.Jewel, showStatus bool, emptyMessage, errPrefix string) error {
	if len(jewels) == 0 {
		fmt.Println(emptyMessage)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := writeJewelTableHeader(w, showStatus); err != nil {
		return fmt.Errorf("%s: write header: %w", errPrefix, err)
	}
	for _, j := range jewels {
		if err := writeJewelTableRow(w, j, showStatus); err != nil {
			return fmt.Errorf("%s: write row: %w", errPrefix, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("%s: flush: %w", errPrefix, err)
	}
	return nil
}

func writeJewelTableHeader(w *tabwriter.Writer, showStatus bool) error {
	var err error
	if showStatus {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "STATUS", "KIND", "TRUST", "SCORE", "STATEMENT")
	} else {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "KIND", "TRUST", "SCORE", "STATEMENT")
	}
	if err != nil {
		return fmt.Errorf("fprintf: %w", err)
	}
	return nil
}

func writeJewelTableRow(w *tabwriter.Writer, j treasure.Jewel, showStatus bool) error {
	var err error
	if showStatus {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Status, j.Kind, j.Trust, j.Score.Value, j.Statement)
	} else {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Kind, j.Trust, j.Score.Value, j.Statement)
	}
	if err != nil {
		return fmt.Errorf("fprintf: %w", err)
	}
	return nil
}

// renderJewelJSON renders a jewel listing as JSON. includeStatus/includeScope
// control which optional fields are populated, matching the two current
// callers: `jewel list` (status, no scope) and `mine list` (status and scope).
func renderJewelJSON(jewels []treasure.Jewel, includeStatus, includeScope bool, errPrefix string) error {
	out := make([]jewelJSONEntry, 0, len(jewels))
	for _, j := range jewels {
		entry := jewelJSONEntry{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
		}
		if includeStatus {
			entry.Status = j.Status
		}
		if includeScope {
			entry.Scope = j.Applicability.Scope
		}
		out = append(out, entry)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("%s: encode json: %w", errPrefix, err)
	}
	return nil
}
