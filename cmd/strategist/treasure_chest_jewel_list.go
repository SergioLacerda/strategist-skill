package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

var validJewelListStatuses = map[string]bool{
	"all":                        true,
	domain.JewelStatusProposed:   true,
	domain.JewelStatusAccepted:   true,
	domain.JewelStatusVerified:   true,
	domain.JewelStatusDeprecated: true,
}

func runTreasureChestJewelList(cmd *cobra.Command, _ []string, opts treasureChestJewelListOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Status = stringFlag(cmd, "status", opts.Status)
	opts.Chest = stringFlag(cmd, "chest", opts.Chest)
	opts.Format = stringFlag(cmd, "format", opts.Format)

	if opts.Status != "" && !validJewelListStatuses[opts.Status] {
		return fmt.Errorf("treasure-chest jewel list: unknown --status %q (want all, proposed, accepted, verified, or deprecated)", opts.Status)
	}

	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest jewel list")
	if err != nil {
		return err
	}

	filtered := treasure.FilterJewels(jewelsByChest, treasure.JewelFilter{
		ChestID: opts.Chest,
		Status:  opts.Status,
	})

	switch opts.Format {
	case "", "table":
		return renderJewelListTable(filtered)
	case "json":
		return renderJewelListJSON(filtered)
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", opts.Format)
	}
}

func renderJewelListTable(jewels []treasure.Jewel) error {
	if len(jewels) == 0 {
		fmt.Println("[Strategist] treasure-chest jewel list: no jewels match the given filters")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "STATUS", "KIND", "TRUST", "SCORE", "STATEMENT"); err != nil {
		return fmt.Errorf("treasure-chest jewel list: write header: %w", err)
	}
	for _, j := range jewels {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Status, j.Kind, j.Trust, j.Score.Value, j.Statement); err != nil {
			return fmt.Errorf("treasure-chest jewel list: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel list: flush: %w", err)
	}
	return nil
}

type jsonJewelListEntry struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Status       string   `json:"status"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
}

func renderJewelListJSON(jewels []treasure.Jewel) error {
	out := make([]jsonJewelListEntry, 0, len(jewels))
	for _, j := range jewels {
		out = append(out, jsonJewelListEntry{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Status:       j.Status,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel list: encode json: %w", err)
	}
	return nil
}
