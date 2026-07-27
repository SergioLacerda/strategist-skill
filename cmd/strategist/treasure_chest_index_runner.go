package main

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

// scanRegisteredChestsForPotions is the "index scans chest content" half of ask #1
// (treasure-chest-cli-unification, absorbing runbook-jewel-relevance-mechanism item
// 5 / SQ-001). Only the "runbooks" chest has a concrete, designed extraction routine
// today (header-extracted when_to_use, see treasure.ScanRunbookDirectory); other
// governed chest kinds ("raw source" -> Jewel candidates) are mentioned in design.md
// as a future extension but have no designed extraction routine yet, so they are
// intentionally skipped rather than guessed at.
func scanRegisteredChestsForPotions(root string, governed map[string]treasure.GovernedChest) ([]treasure.Potion, error) {
	var candidates []treasure.Potion
	for _, chest := range governed {
		if chest.ID != "runbooks" {
			continue
		}
		dirPath := chest.Path
		if !filepath.IsAbs(dirPath) {
			dirPath = filepath.Join(filepath.Dir(root), dirPath)
		}
		found, err := treasure.ScanRunbookDirectory(chest.ID, chest.Trust.Tier, dirPath)
		if err != nil {
			return nil, fmt.Errorf("treasure-chest index: %w", err)
		}
		candidates = append(candidates, found...)
	}
	return candidates, nil
}

func runTreasureChestIndex(root string, rows []treasure.StatusRow, includeHistorical bool) error {
	indexPath := filepath.Join(root, "knowledge.index.yaml")

	if !includeHistorical {
		historical := treasure.HistoricalCount(rows)
		if historical > 0 {
			fmt.Printf("[Strategist] treasure-chest --index: %d historical/lower-trust source(s) excluded from default indexing.\n", historical)
			fmt.Println("             Use --include-historical to opt in.")
		}
	}

	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest --index: %w", err)
	}
	fmt.Printf("[Strategist] treasure-chest --index complete → %s/.compiled/\n", root)
	return nil
}
