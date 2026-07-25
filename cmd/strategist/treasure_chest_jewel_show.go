package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

func runTreasureChestJewelShow(cmd *cobra.Command, args []string, opts treasureChestJewelShowOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)

	id := args[0]
	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest jewel show")
	if err != nil {
		return err
	}

	found, ok := treasure.FindJewel(jewelsByChest, id)
	if !ok {
		return fmt.Errorf("treasure-chest jewel show: jewel %q not found", id)
	}

	switch opts.Format {
	case "", outputFormatTable:
		return renderJewelShowTable(found)
	case outputFormatJSON:
		return renderJewelShowJSON(found)
	default:
		return fmt.Errorf("treasure-chest jewel show: unknown --format %q (want table or json)", opts.Format)
	}
}

func renderJewelShowTable(j treasure.Jewel) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"id", j.ID},
		{"chest_id", j.ChestID},
		{"kind", j.Kind},
		{"status", j.Status},
		{"trust", j.Trust},
		{"statement", j.Statement},
		{"source_refs", fmt.Sprintf("%v", j.SourceRefs)},
		{"reviewed_by", j.ReviewedBy},
		{"last_reviewed", j.LastReviewed},
		{"score.value", fmt.Sprintf("%d", j.Score.Value)},
		{"score.reasons", fmt.Sprintf("%v", j.Score.Reasons)},
		{"applicability.Scope", fmt.Sprintf("%v", j.Applicability.Scope)},
		{"applicability.applies_when", fmt.Sprintf("%v", j.Applicability.AppliesWhen)},
		{"applicability.avoid_when", fmt.Sprintf("%v", j.Applicability.AvoidWhen)},
		{"verification.evidence_refs", fmt.Sprintf("%v", j.Verification.EvidenceRefs)},
		{"history", fmt.Sprintf("%v", j.History)},
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r[0], r[1]); err != nil {
			return fmt.Errorf("treasure-chest jewel show: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel show: flush: %w", err)
	}
	return nil
}

type jsonJewelShowEntry struct {
	ID            string                       `json:"id"`
	ChestID       string                       `json:"chest_id"`
	Kind          string                       `json:"kind"`
	Status        string                       `json:"status"`
	Trust         string                       `json:"trust"`
	Statement     string                       `json:"statement"`
	SourceRefs    []string                     `json:"source_refs"`
	ReviewedBy    string                       `json:"reviewed_by"`
	LastReviewed  string                       `json:"last_reviewed,omitempty"`
	Score         treasure.JewelScore          `json:"score"`
	Applicability treasure.JewelApplicability  `json:"applicability"`
	Verification  treasure.JewelVerification   `json:"verification"`
	History       []treasure.JewelHistoryEntry `json:"history,omitempty"`
}

func renderJewelShowJSON(j treasure.Jewel) error {
	out := jsonJewelShowEntry{
		ID:            j.ID,
		ChestID:       j.ChestID,
		Kind:          j.Kind,
		Status:        j.Status,
		Trust:         j.Trust,
		Statement:     j.Statement,
		SourceRefs:    j.SourceRefs,
		ReviewedBy:    j.ReviewedBy,
		LastReviewed:  j.LastReviewed,
		Score:         j.Score,
		Applicability: j.Applicability,
		Verification:  j.Verification,
		History:       j.History,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel show: encode json: %w", err)
	}
	return nil
}
