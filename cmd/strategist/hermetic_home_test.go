package main

import (
	"os"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
)

// TestMain fails the package run if any test wrote to the invoking user's real
// home directory. Install tests must isolate HOME (setHomeEnv) or skip shim
// writes; without this guard a missing isolation only shows up as a destroyed
// developer installation.
func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithHomeGuard(m))
}
