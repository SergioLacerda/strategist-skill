package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// The wizard classifies every shared LEVELING authority fixture with the same
// reason code as the CLI (see cmd/strategist/leveling_parity_test.go).
func TestWizardLevelingAuthorityParity(t *testing.T) {
	for _, fixture := range testutil.LevelingAuthorityFixtures() {
		t.Run(fixture.Name, func(t *testing.T) {
			root := testutil.PortableStrategistRoot(t)
			fixture.Setup(t, root)
			_, _, err := loadWizardLevelingPolicy(root, newRuntimeDefaultsExtractor(nil))
			assert.Equal(t, fixture.WantCode, testutil.LevelingReasonCode(err), "error: %v", err)
		})
	}
}
