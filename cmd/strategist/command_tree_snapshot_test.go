package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateCommandTree = flag.Bool("update", false, "rewrite testdata/command_tree.golden from the current command tree")

const commandTreeGolden = "testdata/command_tree.golden"

// TestCommandTreeSnapshot pins the path, Use, Short, Long, aliases and flags
// (name, shorthand, type, default, usage) of every command, so help text
// cannot silently drift when a command family moves into an adapter package.
// Regenerate with: go test ./cmd/strategist -run CommandTreeSnapshot -update
func TestCommandTreeSnapshot(t *testing.T) {
	got := renderCommandTree(rootCmd)
	if *updateCommandTree {
		require.NoError(t, os.MkdirAll(filepath.Dir(commandTreeGolden), 0o755))
		require.NoError(t, os.WriteFile(commandTreeGolden, []byte(got), 0o644))
	}
	want, err := os.ReadFile(commandTreeGolden)
	require.NoError(t, err, "missing golden file; run with -update")
	assert.Equal(t, string(want), got, "command tree drifted; if intentional, rerun with -update and review the diff")
}

func renderCommandTree(root *cobra.Command) string {
	var b strings.Builder
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		writeCommandNode(&b, cmd)
		for _, child := range cmd.Commands() {
			if !isIgnoredCommand(child) {
				walk(child)
			}
		}
	}
	walk(root)
	return b.String()
}

func isIgnoredCommand(cmd *cobra.Command) bool {
	// help/completion are injected lazily by Cobra on Execute, so their
	// presence depends on which other tests ran first.
	name := cmd.Name()
	return name == "help" || name == "completion"
}

func writeCommandNode(b *strings.Builder, cmd *cobra.Command) {
	fmt.Fprintf(b, "== %s\n", cmd.CommandPath())
	fmt.Fprintf(b, "use: %s\n", cmd.Use)
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(b, "aliases: %s\n", strings.Join(cmd.Aliases, ", "))
	}
	if cmd.Hidden {
		b.WriteString("hidden: true\n")
	}
	fmt.Fprintf(b, "short: %s\n", cmd.Short)
	if cmd.Long != "" {
		fmt.Fprintf(b, "long: |\n%s\n", indentLines(cmd.Long))
	}
	writeFlags(b, "flag", cmd.LocalNonPersistentFlags())
	writeFlags(b, "persistent", cmd.PersistentFlags())
	b.WriteString("\n")
}

func writeFlags(b *strings.Builder, kind string, flags *pflag.FlagSet) {
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" { // injected lazily, like the help command
			return
		}
		short := ""
		if f.Shorthand != "" {
			short = " -" + f.Shorthand
		}
		fmt.Fprintf(b, "%s: --%s%s (%s) default=%q usage=%q\n", kind, f.Name, short, f.Value.Type(), f.DefValue, f.Usage)
	})
}

func indentLines(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
