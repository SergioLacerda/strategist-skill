package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

func runTreasureChestMineList(root, format string) error {
	// Best-effort: governed is trust-ceiling context only, not required for listing. A
	// corrupt/missing treasure-chests.yaml just means trust ceilings go unchecked here.
	governed, govErr := treasure.LoadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := treasure.LoadJewels(root, governed)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	proposed := treasure.ProposedJewels(jewelsByChest)

	switch format {
	case "", outputFormatTable:
		return renderMineTable(proposed)
	case outputFormatJSON:
		return renderMineJSON(proposed)
	default:
		return fmt.Errorf("treasure-chest mine: unknown --format %q (want table or json)", format)
	}
}

func renderMineTable(jewels []treasure.Jewel) error {
	if len(jewels) == 0 {
		fmt.Println("[Strategist] treasure-chest mine: no proposed jewels awaiting curation")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "KIND", "TRUST", "SCORE", "STATEMENT"); err != nil {
		return fmt.Errorf("treasure-chest mine: write header: %w", err)
	}
	for _, j := range jewels {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Kind, j.Trust, j.Score.Value, j.Statement); err != nil {
			return fmt.Errorf("treasure-chest mine: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest mine: flush: %w", err)
	}
	return nil
}

type jsonMineJewel struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	Status       string   `json:"status"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
	Scope        []string `json:"scope,omitempty"`
}

func renderMineJSON(jewels []treasure.Jewel) error {
	out := make([]jsonMineJewel, 0, len(jewels))
	for _, j := range jewels {
		out = append(out, jsonMineJewel{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			Status:       j.Status,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
			Scope:        j.Applicability.Scope,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest mine: encode json: %w", err)
	}
	return nil
}
