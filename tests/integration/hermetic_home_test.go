//go:build integration

package integration_test

import (
	"os"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
)

// TestMain fails the integration run if any scenario wrote to the invoking
// user's real home directory. The harness isolates HOME per subprocess; this
// guard catches a scenario that bypasses it.
func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithHomeGuard(m))
}
