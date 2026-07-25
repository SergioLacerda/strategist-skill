package main

import (
	"fmt"

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
		return renderJewelTable(proposed, false, "[Strategist] treasure-chest mine: no proposed jewels awaiting curation", "treasure-chest mine")
	case outputFormatJSON:
		return renderJewelJSON(proposed, true, true, "treasure-chest mine")
	default:
		return fmt.Errorf("treasure-chest mine: unknown --format %q (want table or json)", format)
	}
}
