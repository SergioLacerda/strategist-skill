package treasurecli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func treasureChestRootFromCmd(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	root, err := cmd.Flags().GetString(flagRoot)
	if err == nil {
		return root
	}
	root, err = cmd.InheritedFlags().GetString(flagRoot)
	if err == nil {
		return root
	}
	return ""
}

func resolveTreasureChestActionRoot(cmd *cobra.Command, action string) (string, error) {
	prefix := treasureChestActionPrefix(action)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("%s: get cwd: %w", prefix, err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", prefix, err)
	}
	return root, nil
}

func treasureChestActionPrefix(action string) string {
	if strings.HasPrefix(action, "treasure-chest") {
		return action
	}
	return "treasure-chest " + action
}
