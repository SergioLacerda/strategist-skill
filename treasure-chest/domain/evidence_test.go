package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvidenceClassesReturnsStableCopy(t *testing.T) {
	classes := EvidenceClasses()
	assert.Equal(t, []string{
		EvidenceClassExplicit,
		EvidenceClassCorroboratedInference,
		EvidenceClassWeakInference,
		EvidenceClassUnknown,
	}, classes)

	classes[0] = "mutated"
	assert.Equal(t, EvidenceClassExplicit, EvidenceClasses()[0])
}

func TestConfidenceLevelsReturnsStableCopy(t *testing.T) {
	levels := ConfidenceLevels()
	assert.Equal(t, []string{ConfidenceLow, ConfidenceMedium, ConfidenceHigh}, levels)

	levels[0] = "mutated"
	assert.Equal(t, ConfidenceLow, ConfidenceLevels()[0])
}
