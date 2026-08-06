package main

import "github.com/SergioLacerda/strategist-skill/internal/cliutil"

// findStrategistRoot walks up from startDir looking for a .strategist/ directory.
// Delegates to internal/cliutil so internal/treasurecli (and any other importable
// command package) shares the exact same root-discovery behavior as cmd/strategist.
func findStrategistRoot(startDir string) (strategistDir, projectRoot string, err error) {
	return cliutil.FindStrategistRoot(startDir) //nolint:wrapcheck // pure delegation; cliutil's error text is this function's own contract, preserved verbatim on purpose
}

// resolveStrategistRoot returns the runtime root to use for a command.
// Delegates to internal/cliutil — see findStrategistRoot's comment.
func resolveStrategistRoot(explicit, cwd string) (strategistDir, projectRoot string, err error) {
	return cliutil.ResolveStrategistRoot(explicit, cwd) //nolint:wrapcheck // pure delegation, see findStrategistRoot above
}
