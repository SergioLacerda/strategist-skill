package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLevelingConfigEffectiveModeDefaultsToAutomatic(t *testing.T) {
	assert.Equal(t, domain.LevelingModeAutomatic, domain.LevelingConfig{}.EffectiveMode())
	assert.Equal(t, domain.LevelingModeManual, domain.LevelingConfig{Mode: " Manual "}.EffectiveMode())
}

func TestLevelingConfigValidate(t *testing.T) {
	full := map[string]domain.LevelingRoleChoice{"ranger": {Model: "Sonnet", Effort: "high"}}
	cases := []struct {
		name    string
		cfg     domain.LevelingConfig
		wantErr string
	}{
		{"absent block is valid", domain.LevelingConfig{}, ""},
		{"automatic is valid", domain.LevelingConfig{Mode: "automatic"}, ""},
		{"manual with roles is valid", domain.LevelingConfig{Mode: "manual", Roles: full}, ""},
		{"partial role is valid", domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"sniper": {Effort: "low"}}}, ""},
		{"unknown mode", domain.LevelingConfig{Mode: "smart"}, "leveling_mapping_invalid: leveling.mode"},
		{"manual without roles", domain.LevelingConfig{Mode: "manual"}, "leveling_mapping_invalid: leveling.roles"},
		{"unknown role", domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"wizard": {Model: "x", Effort: "low"}}}, "unknown role"},
		{"unknown effort", domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"ranger": {Model: "x", Effort: "turbo"}}}, "effort"},
		{"empty choice", domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"ranger": {}}}, "requires a model or an effort"},
		{"multiline model", domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"ranger": {Model: "a\nb", Effort: "low"}}}, "single-line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestLevelingConfigRoleChoiceCompleteness(t *testing.T) {
	cfg := domain.LevelingConfig{Mode: "manual", Roles: map[string]domain.LevelingRoleChoice{"Ranger": {Model: "Sonnet", Effort: "high"}, "sniper": {Model: "Haiku"}}}
	choice, ok := cfg.Choice("ranger")
	require.True(t, ok)
	assert.True(t, choice.Complete())
	partial, ok := cfg.Choice("sniper")
	require.True(t, ok)
	assert.False(t, partial.Complete())
	_, ok = cfg.Choice("scout")
	assert.False(t, ok)
	_, ok = domain.LevelingConfig{Mode: "automatic", Roles: cfg.Roles}.Choice("ranger")
	assert.False(t, ok, "automatic mode never exposes manual choices")
}

func TestActiveConfigValidatesLevelingBlock(t *testing.T) {
	raw := "mode: epic\nbase_path: .analysis\nslots: {discovery: a, refinement: b, execution: c}\nleveling:\n  mode: manual\n  roles:\n    ranger: {model: Sonnet, effort: turbo}\n"
	var cfg domain.ActiveConfig
	require.NoError(t, yaml.Unmarshal([]byte(raw), &cfg))
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leveling_mapping_invalid")
}

func TestActiveConfigWithoutLevelingBlockStillValid(t *testing.T) {
	raw := "mode: epic\nbase_path: .analysis\nslots: {discovery: a, refinement: b, execution: c}\n"
	var cfg domain.ActiveConfig
	require.NoError(t, yaml.Unmarshal([]byte(raw), &cfg))
	assert.NoError(t, cfg.Validate())
}
