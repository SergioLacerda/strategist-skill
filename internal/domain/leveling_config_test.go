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

func TestLevelingConfigManualIsHostPassthrough(t *testing.T) {
	assert.True(t, domain.LevelingConfig{Mode: "manual"}.HostPassthrough())
	assert.False(t, domain.LevelingConfig{Mode: "automatic"}.HostPassthrough())
	assert.False(t, domain.LevelingConfig{}.HostPassthrough(), "an absent block is automatic")
}

func TestLevelingConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     domain.LevelingConfig
		wantErr string
	}{
		{"absent block is valid", domain.LevelingConfig{}, ""},
		{"automatic is valid", domain.LevelingConfig{Mode: "automatic"}, ""},
		{"manual needs nothing else", domain.LevelingConfig{Mode: "manual"}, ""},
		{"unknown mode", domain.LevelingConfig{Mode: "smart"}, "leveling_mapping_invalid: leveling.mode"},
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

func TestActiveConfigValidatesLevelingBlock(t *testing.T) {
	raw := "mode: epic\nbase_path: .analysis\nslots: {discovery: a, refinement: b, execution: c}\nleveling:\n  mode: smart\n"
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
