package treasurecli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

func runTreasureChestItemsShow(cmd *cobra.Command, args []string, opts treasureChestItemsShowOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)
	id := args[0]

	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest items show")
	if err != nil {
		return err
	}
	if j, ok := treasure.FindJewel(jewelsByChest, id); ok {
		return showJewelItem(j, opts.Format)
	}

	potionsByChest, err := loadPotionsForCmd(cmd, "treasure-chest items show")
	if err != nil {
		return err
	}
	if p, ok := treasure.FindPotion(potionsByChest, id); ok {
		return showPotionItem(p, opts.Format)
	}

	return fmt.Errorf("treasure-chest items show: item %q not found (checked jewels and potions)", id)
}

func showJewelItem(j treasure.Jewel, format string) error {
	switch format {
	case "", outputFormatTable:
		return renderJewelShowTable(j)
	case outputFormatJSON:
		return renderJewelShowJSON(j)
	default:
		return fmt.Errorf("treasure-chest items show: unknown --format %q (want table or json)", format)
	}
}

func showPotionItem(p treasure.Potion, format string) error {
	switch format {
	case "", outputFormatTable:
		return renderPotionShowTable(p)
	case outputFormatJSON:
		return renderPotionShowJSON(p)
	default:
		return fmt.Errorf("treasure-chest items show: unknown --format %q (want table or json)", format)
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
			return fmt.Errorf("treasure-chest items show: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest items show: flush: %w", err)
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
		return fmt.Errorf("treasure-chest items show: encode json: %w", err)
	}
	return nil
}

func renderPotionShowTable(p treasure.Potion) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"id", p.ID},
		{"chest_id", p.ChestID},
		{"runbook_ref", p.RunbookRef},
		{"status", p.Status},
		{"trust", p.Trust},
		{"when_to_use", p.WhenToUse},
		{"when_to_avoid", p.WhenToAvoid},
		{"source_refs", fmt.Sprintf("%v", p.SourceRefs)},
		{"reviewed_by", p.ReviewedBy},
		{"last_reviewed", p.LastReviewed},
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r[0], r[1]); err != nil {
			return fmt.Errorf("treasure-chest items show: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest items show: flush: %w", err)
	}
	return nil
}

type jsonPotionShowEntry struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	RunbookRef   string   `json:"runbook_ref"`
	Status       string   `json:"status"`
	Trust        string   `json:"trust"`
	WhenToUse    string   `json:"when_to_use"`
	WhenToAvoid  string   `json:"when_to_avoid,omitempty"`
	SourceRefs   []string `json:"source_refs"`
	ReviewedBy   string   `json:"reviewed_by"`
	LastReviewed string   `json:"last_reviewed,omitempty"`
}

func renderPotionShowJSON(p treasure.Potion) error {
	out := jsonPotionShowEntry{
		ID:           p.ID,
		ChestID:      p.ChestID,
		RunbookRef:   p.RunbookRef,
		Status:       p.Status,
		Trust:        p.Trust,
		WhenToUse:    p.WhenToUse,
		WhenToAvoid:  p.WhenToAvoid,
		SourceRefs:   p.SourceRefs,
		ReviewedBy:   p.ReviewedBy,
		LastReviewed: p.LastReviewed,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest items show: encode json: %w", err)
	}
	return nil
}
