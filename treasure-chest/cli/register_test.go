package treasurecli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegister_AttachesTreasureChestAndRunbookCommands(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "strategist"}
	Register(root)

	var names []string
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	assert.Contains(t, names, treasureChestCmd.Name())
	assert.Contains(t, names, runbookCmd.Name())
}
