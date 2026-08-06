package treasurecli

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

func runTreasureChestRemove(cmd *cobra.Command, args []string, opts treasureChestRemoveOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.ID = stringFlag(cmd, "id", opts.ID)

	pathArg := optionalPathArg(args)
	if pathArg == "" && opts.ID == "" {
		return fmt.Errorf("treasure-chest remove: provide a path or --id")
	}

	root, err := resolveTreasureChestActionRoot(cmd, "remove")
	if err != nil {
		return err
	}

	id, hasJewels, err := applyTreasureChestRemove(root, pathArg, opts.ID)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}
	reportRemoveResult(id, hasJewels)
	return nil
}

func applyTreasureChestRemove(root, pathArg, idOpt string) (string, bool, error) {
	id, err := treasure.ResolveRemoveTarget(root, pathArg, idOpt)
	if err != nil {
		return "", false, fmt.Errorf("resolve remove target: %w", err)
	}
	paths := treasure.NewChestPaths(root)
	docs, err := treasure.LoadRemoveDocs(paths)
	if err != nil {
		return "", false, fmt.Errorf("load remove docs: %w", err)
	}
	if err := treasure.ApplyRemoveMutations(docs, id); err != nil {
		return "", false, fmt.Errorf("apply remove mutations: %w", err)
	}
	written, err := treasure.WriteRemoveDocs(paths, docs)
	if err != nil {
		return "", false, fmt.Errorf("partial write after %v: %w", written, err)
	}
	refreshConfigLock(root, paths.Active)
	return id, len(docs.Jewels) > 0, nil
}

func optionalPathArg(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return ""
}

func reportRemoveResult(id string, hasJewels bool) {
	fmt.Printf("[Strategist] remove: OK (id=%s)\n", id)
	if hasJewels {
		fmt.Printf("[Strategist] remove: %q's jewels marked deprecated in jewels.yaml\n", id)
	}
	fmt.Println("[Strategist] remove: index is stale. Run: strategist treasure-chest index")
}
