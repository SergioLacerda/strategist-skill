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
//
// It also keeps the run hermetic for the provider integration: the developer's
// provider credentials are scrubbed from the environment, and the provider
// transport defaults to a guard that refuses every request, so a test that
// forgets to inject a simulated provider fails loudly instead of calling out.
func TestMain(m *testing.M) {
	delegationRoundTripper = networkGuard
	code := testutil.RunHermetic(m)
	cleanupUpgradeTemplate()
	os.Exit(code)
}

// networkGuard is the default provider transport of this test binary.
var networkGuard = testutil.NewDenyNetwork()
