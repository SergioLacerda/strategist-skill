package main

import (
	"fmt"

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
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)

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
	case "", outputFormatTable:
		return renderJewelTable(filtered, true, "[Strategist] treasure-chest jewel list: no jewels match the given filters", "treasure-chest jewel list")
	case outputFormatJSON:
		return renderJewelJSON(filtered, true, false, "treasure-chest jewel list")
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", opts.Format)
	}
}
