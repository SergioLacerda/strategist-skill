package leveling

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

// The manual LEVELING choice (active.yaml) validates efforts against the domain
// catalog; it must stay identical to the policy's tier catalog.
func TestManualEffortTiersMatchPolicyCatalog(t *testing.T) {
	assert.Len(t, effortTiers, len(domain.LevelingEffortTiers))
	for _, tier := range domain.LevelingEffortTiers {
		assert.True(t, effortTiers[tier], tier)
	}
}
