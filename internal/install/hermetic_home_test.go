package install

import (
	"os"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
)

// TestMain fails the package run if any test wrote to the invoking user's real
// home directory. The shim step resolves os.UserHomeDir() directly, so a test
// that forgets to isolate HOME overwrites the developer's installation.
func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithHomeGuard(m))
}
