package check

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegister_AttachesCheckAndCheckStaleCommands(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "strategist"}
	Register(root)

	var names []string
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	assert.Contains(t, names, checkCmd.Name())
	assert.Contains(t, names, checkStaleCmd.Name())
}
