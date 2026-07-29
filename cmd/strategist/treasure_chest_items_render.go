package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

// itemRow is the unified shape `items list`/`items show` render — a Jewel and a
// Potion are different candidate types with different field sets, but both are
// "items" at the CLI surface (see Decision D5,
// .analysis/refined/treasure-chest-cli-unification/analysis.md).
type itemRow struct {
	Kind       string // "jewel" or "potion"
	ID         string
	ChestID    string
	Status     string
	Trust      string
	ScoreValue int    // jewel only; zero for potions
	Summary    string // jewel.Statement or potion.WhenToUse
	SourceRefs []string
}

func jewelToItemRow(j treasure.Jewel) itemRow {
	return itemRow{
		Kind:       "jewel",
		ID:         j.ID,
		ChestID:    j.ChestID,
		Status:     j.Status,
		Trust:      j.Trust,
		ScoreValue: j.Score.Value,
		Summary:    j.Statement,
		SourceRefs: j.SourceRefs,
	}
}

func potionToItemRow(p treasure.Potion) itemRow {
	return itemRow{
		Kind:       "potion",
		ID:         p.ID,
		ChestID:    p.ChestID,
		Status:     p.Status,
		Trust:      p.Trust,
		Summary:    p.WhenToUse,
		SourceRefs: p.SourceRefs,
	}
}

// sortItemRows orders items by chest id, then kind, then item id — deterministic
// output when `items list` merges jewels and potions.
func sortItemRows(rows []itemRow) {
	sort.Slice(rows, func(i, k int) bool {
		if rows[i].ChestID != rows[k].ChestID {
			return rows[i].ChestID < rows[k].ChestID
		}
		if rows[i].Kind != rows[k].Kind {
			return rows[i].Kind < rows[k].Kind
		}
		return rows[i].ID < rows[k].ID
	})
}

func renderItemTable(rows []itemRow, emptyMessage, errPrefix string) error {
	if len(rows) == 0 {
		fmt.Println(emptyMessage)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "KIND", "STATUS", "TRUST", "SCORE", "SUMMARY"); err != nil {
		return fmt.Errorf("%s: write header: %w", errPrefix, err)
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n", r.ID, r.ChestID, r.Kind, r.Status, r.Trust, r.ScoreValue, r.Summary); err != nil {
			return fmt.Errorf("%s: write row: %w", errPrefix, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("%s: flush: %w", errPrefix, err)
	}
	return nil
}

type itemJSONEntry struct {
	ID         string   `json:"id"`
	ChestID    string   `json:"chest_id"`
	Kind       string   `json:"kind"`
	Status     string   `json:"status"`
	Trust      string   `json:"trust"`
	ScoreValue int      `json:"score_value,omitempty"`
	Summary    string   `json:"summary"`
	SourceRefs []string `json:"source_refs"`
}

func renderItemJSON(rows []itemRow, errPrefix string) error {
	out := make([]itemJSONEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, itemJSONEntry{
			ID:         r.ID,
			ChestID:    r.ChestID,
			Kind:       r.Kind,
			Status:     r.Status,
			Trust:      r.Trust,
			ScoreValue: r.ScoreValue,
			Summary:    r.Summary,
			SourceRefs: r.SourceRefs,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("%s: encode json: %w", errPrefix, err)
	}
	return nil
}
