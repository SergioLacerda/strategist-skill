package plugins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalRoot writes the smallest .strategist/ root evaluate-write and
// authorize can resolve: an active.yaml with a base_path.
func minimalRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: pragmatic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	return dir
}

// TestRegister_CommandTreeParity pins the public command surface: subcommand
// names/order and every flag name with its default.
func TestRegister_CommandTreeParity(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "strategist"}
	Register(root)

	parent, _, err := root.Find([]string{"plugins"})
	require.NoError(t, err)
	require.Equal(t, "plugins", parent.Name())

	var names []string
	for _, sub := range parent.Commands() {
		names = append(names, sub.Name())
	}
	assert.Equal(t, []string{"authorize", "evaluate-write", "prepare-embedded"}, names)

	want := map[string]map[string]string{
		"authorize": {
			"root": "", "target": "", "mission": "", "approval-gate": "pending",
			"execution-gate": "allowed", "json": "false",
		},
		"evaluate-write": {"root": "", "target": ""},
		"prepare-embedded": {
			"source": "external-skills-source", "defaults-root": filepath.Join("internal", "embed", "defaults"),
			"lock": "external-skills-source.lock.yaml", "check": "false",
		},
	}
	for _, sub := range parent.Commands() {
		got := map[string]string{}
		sub.Flags().VisitAll(func(f *pflag.Flag) { got[f.Name] = f.DefValue })
		assert.Equal(t, want[sub.Name()], got, "flags of %s", sub.Name())
	}
}

func TestNew_ReturnsIsolatedTrees(t *testing.T) {
	t.Parallel()
	first, second := New(), New()
	require.NoError(t, first.Commands()[0].Flags().Set("target", "docs/a.md"))
	assert.Empty(t, second.Commands()[0].Flags().Lookup("target").Value.String())
}

func runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "strategist", SilenceUsage: true, SilenceErrors: true}
	Register(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}
