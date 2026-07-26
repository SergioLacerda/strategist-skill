package i18n_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestReservedGateTokensPTBR(t *testing.T) {
	t.Parallel()

	tokens := i18n.ReservedGateTokensPTBR()
	assert.ElementsMatch(t, []string{
		i18n.ReservedGateYes,
		i18n.ReservedGateNo,
		i18n.ReservedGateAccept,
		i18n.ReservedGateRevisionRequested,
		i18n.ReservedGateReject,
	}, tokens)

	tokens[0] = "mutated"
	assert.Equal(t, i18n.ReservedGateYes, i18n.ReservedGateTokensPTBR()[0])
}

func TestReservedQuickDrawTokens(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{
		i18n.ReservedQuickDrawPT,
		i18n.ReservedQuickDrawEN,
	}, i18n.ReservedQuickDrawTokens())
}
