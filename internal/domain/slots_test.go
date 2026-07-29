package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRequiredSlots(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []domain.SlotName{
		domain.SlotDiscovery,
		domain.SlotRefinement,
		domain.SlotExecution,
	}, domain.RequiredSlots())
}

func TestIsValidSlot(t *testing.T) {
	t.Parallel()

	assert.True(t, domain.IsValidSlot("discovery"))
	assert.True(t, domain.IsValidSlot("refinement"))
	assert.True(t, domain.IsValidSlot("execution"))
	assert.False(t, domain.IsValidSlot("bogus"))
}
