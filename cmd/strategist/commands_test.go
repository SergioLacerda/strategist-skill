package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRootCmd_RegistersEveryTopLevelCommand pins the set of top-level commands
// independently of the help-text snapshot, so moving registration around (see
// registerCommands) can never silently drop or duplicate a command.
func TestRootCmd_RegistersEveryTopLevelCommand(t *testing.T) {
	want := []string{
		"check", "check-stale", "compile", "dojo", "eval", "handoff", "install",
		"leveling", "mechanisms", "metrics", "mission", "plugins", "provider", "runbook", "sync-governance",
		"treasure-chest", "upgrade", "validate", "version",
	}

	var got []string
	for _, c := range rootCmd.Commands() {
		if isIgnoredCommand(c) {
			continue
		}
		got = append(got, c.Name())
	}
	sort.Strings(got)

	assert.Equal(t, want, got)
}

func TestRootCmd_TopLevelCommandsAreRegisteredOnce(t *testing.T) {
	seen := map[string]int{}
	for _, c := range rootCmd.Commands() {
		seen[c.Name()]++
	}
	for name, n := range seen {
		assert.Equal(t, 1, n, name)
	}
}
