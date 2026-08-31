package domain_test

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestEvaluatePromotionPacketAging_BeforeThresholdIsNotAged(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	today := createdAt.AddDate(0, 0, domain.PromotionPacketAgingThresholdDays-1)

	decision := domain.EvaluatePromotionPacketAging(createdAt, today)

	assert.False(t, decision.Aged)
	assert.Equal(t, domain.PromotionPacketAgingThresholdDays-1, decision.DaysOpen)
}

func TestEvaluatePromotionPacketAging_AtThresholdIsAged(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	today := createdAt.AddDate(0, 0, domain.PromotionPacketAgingThresholdDays)

	decision := domain.EvaluatePromotionPacketAging(createdAt, today)

	assert.True(t, decision.Aged)
	assert.Equal(t, domain.PromotionPacketAgingThresholdDays, decision.DaysOpen)
}

func TestEvaluatePromotionPacketAging_AfterThresholdIsAged(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	today := createdAt.AddDate(0, 0, domain.PromotionPacketAgingThresholdDays+15)

	decision := domain.EvaluatePromotionPacketAging(createdAt, today)

	assert.True(t, decision.Aged)
	assert.Equal(t, domain.PromotionPacketAgingThresholdDays+15, decision.DaysOpen)
}

func TestEvaluatePromotionPacketAging_SameDayIsNotAged(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	today := createdAt.Add(2 * time.Hour)

	decision := domain.EvaluatePromotionPacketAging(createdAt, today)

	assert.False(t, decision.Aged)
	assert.Equal(t, 0, decision.DaysOpen)
}

func TestPromotionPacket_ZeroValueStatusIsEmptyNotPending(t *testing.T) {
	t.Parallel()
	// The zero value must not silently look like a deliberate "pending"
	// status — callers that construct a PromotionPacket are expected to
	// set Status explicitly (mirrors jewels/potions never leaving status
	// implicit; see ADR-0012).
	var p domain.PromotionPacket
	assert.Empty(t, p.Status)
	assert.NotEqual(t, domain.PromotionPacketStatusPending, p.Status)
}
