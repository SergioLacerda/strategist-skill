package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

var validItemListStatuses = map[string]bool{
	"all":                        true,
	domain.JewelStatusProposed:   true,
	domain.JewelStatusAccepted:   true,
	domain.JewelStatusVerified:   true,
	domain.JewelStatusDeprecated: true,
}

var validItemListKinds = map[string]bool{
	"":       true, // both
	"jewel":  true,
	"potion": true,
}

func runTreasureChestItemsList(cmd *cobra.Command, _ []string, opts treasureChestItemsListOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Kind = stringFlag(cmd, "kind", opts.Kind)
	opts.Status = stringFlag(cmd, "status", opts.Status)
	opts.Chest = stringFlag(cmd, "chest", opts.Chest)
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)

	if !validItemListKinds[opts.Kind] {
		return fmt.Errorf("treasure-chest items list: unknown --kind %q (want jewel or potion)", opts.Kind)
	}
	if opts.Status != "" && !validItemListStatuses[opts.Status] {
		return fmt.Errorf("treasure-chest items list: unknown --status %q (want all, proposed, accepted, verified, or deprecated)", opts.Status)
	}

	rows, err := loadItemRows(cmd, opts)
	if err != nil {
		return err
	}

	switch opts.Format {
	case "", outputFormatTable:
		return renderItemTable(rows, "[Strategist] treasure-chest items list: no items match the given filters", "treasure-chest items list")
	case outputFormatJSON:
		return renderItemJSON(rows, "treasure-chest items list")
	default:
		return fmt.Errorf("treasure-chest items list: unknown --format %q (want table or json)", opts.Format)
	}
}

func loadItemRows(cmd *cobra.Command, opts treasureChestItemsListOptions) ([]itemRow, error) {
	var rows []itemRow

	if includeItemKind(opts.Kind, "jewel") {
		jewelRows, err := loadJewelItemRows(cmd, opts)
		if err != nil {
			return nil, err
		}
		rows = append(rows, jewelRows...)
	}

	if includeItemKind(opts.Kind, "potion") {
		potionRows, err := loadPotionItemRows(cmd, opts)
		if err != nil {
			return nil, err
		}
		rows = append(rows, potionRows...)
	}

	sortItemRows(rows)
	return rows, nil
}

// includeItemKind reports whether `list` should include rows of the given kind:
// an empty --kind means "both", otherwise only an exact match is included.
func includeItemKind(kind, want string) bool {
	return kind == "" || kind == want
}

func loadJewelItemRows(cmd *cobra.Command, opts treasureChestItemsListOptions) ([]itemRow, error) {
	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest items list")
	if err != nil {
		return nil, err
	}
	filtered := treasure.FilterJewels(jewelsByChest, treasure.JewelFilter{ChestID: opts.Chest, Status: opts.Status})
	rows := make([]itemRow, 0, len(filtered))
	for _, j := range filtered {
		rows = append(rows, jewelToItemRow(j))
	}
	return rows, nil
}

func loadPotionItemRows(cmd *cobra.Command, opts treasureChestItemsListOptions) ([]itemRow, error) {
	potionsByChest, err := loadPotionsForCmd(cmd, "treasure-chest items list")
	if err != nil {
		return nil, err
	}
	filtered := treasure.FilterPotions(potionsByChest, treasure.PotionFilter{ChestID: opts.Chest, Status: opts.Status})
	rows := make([]itemRow, 0, len(filtered))
	for _, p := range filtered {
		rows = append(rows, potionToItemRow(p))
	}
	return rows, nil
}
