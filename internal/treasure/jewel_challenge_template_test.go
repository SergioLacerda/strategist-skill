package treasure

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func baseJewel(kind string) Jewel {
	return Jewel{
		ID:         "jewel-handoff-007",
		ChestID:    "handoff",
		Kind:       kind,
		Statement:  "A recommendation was silently promoted to a requirement across a handoff.",
		SourceRefs: []string{"docs/incident-2026-08-01.md"},
		Status:     domain.JewelStatusAccepted,
		Score:      JewelScore{Value: 80},
	}
}

func TestValidateJewelEntry_ChallengeTemplateFieldsRequireKindTemplate(t *testing.T) {
	t.Parallel()

	j := baseJewel("pattern")
	j.Pattern = "recommendation_promoted_to_requirement"

	err := ValidateJewelEntry(j, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "challenge_template")
}

func TestValidateJewelEntry_ChallengeTemplateFieldsAllowedOnKindTemplate(t *testing.T) {
	t.Parallel()

	j := baseJewel(jewelKindTemplate)
	j.Pattern = "recommendation_promoted_to_requirement"
	j.Severity = "high"
	j.ChallengeTemplate = &ChallengeTemplate{
		AppliesTo: []string{"archivist_to_sniper", "ranger_to_archivist"},
		Type:      "classification",
		Prompt:    "Is this a decision you're required to make, or a recommendation you're free to accept or reject?",
	}

	assert.NoError(t, ValidateJewelEntry(j, nil))
}

func TestValidateJewelEntry_ExistingTemplateJewelWithoutChallengeFieldsStillValid(t *testing.T) {
	t.Parallel()

	// Additive-field round-trip: a pre-existing kind:template jewel with none
	// of the new fields set must keep parsing/validating unchanged.
	j := baseJewel(jewelKindTemplate)
	assert.NoError(t, ValidateJewelEntry(j, nil))
}

func TestValidateJewelEntry_InvalidSeverity(t *testing.T) {
	t.Parallel()

	j := baseJewel(jewelKindTemplate)
	j.Severity = "urgent"

	err := ValidateJewelEntry(j, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "severity")
}

func TestValidateJewelEntry_ChallengeTemplateMissingRequiredSubfields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ct   ChallengeTemplate
		want string
	}{
		{"missing applies_to", ChallengeTemplate{Type: "classification", Prompt: "p"}, "applies_to"},
		{"missing type", ChallengeTemplate{AppliesTo: []string{"archivist_to_sniper"}, Prompt: "p"}, "type"},
		{"invalid type", ChallengeTemplate{AppliesTo: []string{"archivist_to_sniper"}, Type: "essay", Prompt: "p"}, "not an allowed value"},
		{"missing prompt", ChallengeTemplate{AppliesTo: []string{"archivist_to_sniper"}, Type: "classification"}, "prompt"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			j := baseJewel(jewelKindTemplate)
			j.ChallengeTemplate = &tc.ct
			err := ValidateJewelEntry(j, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestJewel_ChallengeTemplateYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	j := baseJewel(jewelKindTemplate)
	j.Pattern = "recommendation_promoted_to_requirement"
	j.Severity = "high"
	j.ChallengeTemplate = &ChallengeTemplate{
		AppliesTo: []string{"archivist_to_sniper"},
		Type:      "classification",
		Prompt:    "Is this a decision or a recommendation?",
	}

	out, err := yaml.Marshal(j)
	require.NoError(t, err)

	var roundTripped Jewel
	require.NoError(t, yaml.Unmarshal(out, &roundTripped))

	assert.Equal(t, j.Pattern, roundTripped.Pattern)
	assert.Equal(t, j.Severity, roundTripped.Severity)
	require.NotNil(t, roundTripped.ChallengeTemplate)
	assert.Equal(t, *j.ChallengeTemplate, *roundTripped.ChallengeTemplate)
}

func TestJewel_YAMLWithoutChallengeTemplateFieldsOmitsThem(t *testing.T) {
	t.Parallel()

	// Existing manifests with no challenge-template fields must serialize
	// with no trace of the new keys — additive, not just optional.
	j := baseJewel("decision")

	out, err := yaml.Marshal(j)
	require.NoError(t, err)

	assert.NotContains(t, string(out), "pattern:")
	assert.NotContains(t, string(out), "challenge_template:")
	assert.NotContains(t, string(out), "severity:")
}
