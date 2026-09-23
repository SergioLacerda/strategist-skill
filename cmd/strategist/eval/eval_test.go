package eval

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopSilenceRun(*cobra.Command) {}

// TestRegister_CommandTreeParity pins the public command surface: subcommand
// names/order and every flag name with its default, mirroring
// plugins_adapter_test.go/metrics_adapter_test.go's own parity test.
func TestRegister_CommandTreeParity(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "strategist"}
	Register(root, Dependencies{RootFlag: "root", SilenceRun: noopSilenceRun}, HarvestDependencies{RootFlag: "root", SilenceRun: noopSilenceRun})

	parent, _, err := root.Find([]string{"eval"})
	require.NoError(t, err)
	require.Equal(t, "eval", parent.Name())

	var names []string
	for _, sub := range parent.Commands() {
		names = append(names, sub.Name())
	}
	assert.Equal(t, []string{"harvest", "run"}, names)

	want := map[string]map[string]string{
		"run":     {"race": "true", "root": ""},
		"harvest": {"all": "false", "include": "", "root": ""},
	}
	for _, sub := range parent.Commands() {
		got := map[string]string{}
		sub.Flags().VisitAll(func(f *pflag.Flag) { got[f.Name] = f.DefValue })
		assert.Equal(t, want[sub.Name()], got, "flags of %s", sub.Name())
	}
}

// TestNewRun_ReturnsIsolatedCommands guards against reintroducing a
// package-level command var (the bug class eval_run.go's evalRunCmd global
// used to risk): two independently constructed commands must not share flag
// state.
func TestNewRun_ReturnsIsolatedCommands(t *testing.T) {
	t.Parallel()
	first := NewRun(Dependencies{RootFlag: "root", SilenceRun: noopSilenceRun})
	second := NewRun(Dependencies{RootFlag: "root", SilenceRun: noopSilenceRun})
	require.NoError(t, first.Flags().Set("root", "/first"))
	assert.Empty(t, second.Flags().Lookup("root").Value.String())
}
