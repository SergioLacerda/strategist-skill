package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveConfig_ValidateNoLegacyFields(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.ActiveConfig{}.ValidateNoLegacyFields())

	err := domain.ActiveConfig{ExecutionMode: "sync"}.ValidateNoLegacyFields()
	require.ErrorContains(t, err, "execution_mode")

	err = domain.ActiveConfig{GitPersistenceMode: "auto"}.ValidateNoLegacyFields()
	require.ErrorContains(t, err, "git_persistence_mode")
}

func TestPersonaConfig_ValidateForRuntime(t *testing.T) {
	t.Parallel()
	valid := domain.PersonaConfig{
		ID:            "ranger",
		ToneDirective: "focused",
		PhaseLabels: domain.PhaseLabels{
			Discovery:  "Discovery",
			Refinement: "Refinement",
			Execution:  "Execution",
		},
		Diagnostics: domain.PersonaDiagnostics{
			PipelineHeader:  "header",
			BootstrapOrigin: "origin",
		},
	}
	require.NoError(t, valid.ValidateForRuntime())

	err := domain.PersonaConfig{}.ValidateForRuntime()
	require.ErrorContains(t, err, "id is required")
	require.ErrorContains(t, err, "tone_directive is required")
	require.ErrorContains(t, err, "phase_labels")
	require.ErrorContains(t, err, "diagnostics.pipeline_header")
	require.ErrorContains(t, err, "diagnostics.bootstrap_origin")
}

func TestRoleConfig_Validate(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.RoleConfig{Role: "sniper", Slot: "execution"}.Validate())

	err := domain.RoleConfig{}.Validate()
	require.ErrorContains(t, err, "role is required")
	require.ErrorContains(t, err, "slot is required")

	err = domain.RoleConfig{Role: "sniper", Slot: "bogus"}.Validate()
	require.ErrorContains(t, err, `slot "bogus" is not one of`)
}

func TestRoleSlotMap_Validate(t *testing.T) {
	t.Parallel()
	full := domain.RoleSlotMap{"discovery": "ranger", "refinement": "archivist", "execution": "sniper"}
	require.NoError(t, full.Validate())

	err := domain.RoleSlotMap{}.Validate()
	require.ErrorContains(t, err, "missing slot: discovery")
	require.ErrorContains(t, err, "missing slot: refinement")
	require.ErrorContains(t, err, "missing slot: execution")
}

func TestActiveConfig_Validate(t *testing.T) {
	t.Parallel()
	valid := domain.ActiveConfig{
		Mode:     "epic",
		BasePath: ".analysis",
		Slots:    map[string]string{"discovery": "ranger"},
	}
	require.NoError(t, valid.Validate())

	err := domain.ActiveConfig{}.Validate()
	require.ErrorContains(t, err, "mode is required")
	require.ErrorContains(t, err, "base_path is required")
	require.ErrorContains(t, err, "slots must have at least one entry")
}

func TestPersonaConfig_Validate(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.PersonaConfig{ID: "ranger", ToneDirective: "focused"}.Validate())

	err := domain.PersonaConfig{}.Validate()
	require.ErrorContains(t, err, "id is required")
	require.ErrorContains(t, err, "tone_directive is required")
}

func TestDojoCheckResult_PassedAndFailCount(t *testing.T) {
	t.Parallel()
	allPassed := domain.DojoCheckResult{Items: []domain.DojoCheckItem{{Passed: true}, {Passed: true}}}
	assert.True(t, allPassed.Passed())
	assert.Equal(t, 0, allPassed.FailCount())

	mixed := domain.DojoCheckResult{Items: []domain.DojoCheckItem{{Passed: true}, {Passed: false}, {Passed: false}}}
	assert.False(t, mixed.Passed())
	assert.Equal(t, 2, mixed.FailCount())
}
