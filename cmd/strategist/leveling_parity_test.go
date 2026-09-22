package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The CLI classifies every shared LEVELING authority fixture with the same
// reason code as the wizard (see internal/install/wizard_leveling_parity_test.go).
func TestCLILevelingAuthorityParity(t *testing.T) {
	for _, fixture := range testutil.LevelingAuthorityFixtures() {
		t.Run(fixture.Name, func(t *testing.T) {
			root := testutil.PortableStrategistRoot(t)
			fixture.Setup(t, root)
			oldCWD, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(filepath.Dir(root)))
			t.Cleanup(func() { _ = os.Chdir(oldCWD) })

			_, _, err = loadLevelingPolicy()
			assert.Equal(t, fixture.WantCode, testutil.LevelingReasonCode(err), "error: %v", err)
		})
	}
}
