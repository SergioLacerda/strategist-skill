package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Error sentinels ---

func TestErrors_NotNil(t *testing.T) {
	t.Parallel()
	require.Error(t, domain.ErrArtifactAbsent)
	require.Error(t, domain.ErrManifestMissing)
	require.Error(t, domain.ErrSourceStale)
}

func TestErrors_Messages(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "artifact does not exist", domain.ErrArtifactAbsent.Error())
	assert.Equal(t, "manifest not found", domain.ErrManifestMissing.Error())
	assert.Equal(t, "source file modified after artifact", domain.ErrSourceStale.Error())
}

func TestErrors_AreDistinct(t *testing.T) {
	t.Parallel()
	errs := []error{domain.ErrArtifactAbsent, domain.ErrManifestMissing, domain.ErrSourceStale}
	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				assert.ErrorIs(t, a, b)
			} else {
				assert.NotErrorIs(t, a, b, "errors[%d] and errors[%d] must be distinct", i, j)
			}
		}
	}
}

// --- Struct JSON tags ---

func TestCompiledConfig_JSONTags(t *testing.T) {
	t.Parallel()
	cc := domain.CompiledConfig{
		Schema:     "strategist-compiled-config/1.0",
		CompiledAt: "123",
		Sources:    map[string]int64{"/a": 100},
		Active:     map[string]any{"mode": "full"},
		Personas:   map[string]any{"epic": "yes"},
		Roles:      map[string]any{"default": "yes"},
	}
	data, err := json.Marshal(cc)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"schema"`)
	assert.Contains(t, s, `"compiled_at"`)
	assert.Contains(t, s, `"sources"`)
	assert.Contains(t, s, `"active"`)
	assert.Contains(t, s, `"personas"`)
	assert.Contains(t, s, `"roles"`)
}

func TestCompiledDomain_JSONTags(t *testing.T) {
	t.Parallel()
	cd := domain.CompiledDomain{Schema: "d/1.0", CompiledAt: "t", Sources: nil, Domain: nil}
	data, err := json.Marshal(cd)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema"`)
}

func TestCompiledIndex_JSONTags(t *testing.T) {
	t.Parallel()
	ci := domain.CompiledIndex{Schema: "i/1.0"}
	data, err := json.Marshal(ci)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema"`)
}

func TestCompiledManifest_JSONTags(t *testing.T) {
	t.Parallel()
	cm := domain.CompiledManifest{
		Schema:    "m/1.0",
		Artifacts: map[string]string{".config.gz": "sha256:abc"},
	}
	data, err := json.Marshal(cm)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"artifacts"`)
}

func TestInstallConfig_Fields(t *testing.T) {
	t.Parallel()
	cfg := domain.InstallConfig{Target: "/tmp/x", Silent: true, Wizard: false}
	assert.Equal(t, "/tmp/x", cfg.Target)
	assert.True(t, cfg.Silent)
	assert.False(t, cfg.Wizard)
}

func TestWizardConfig_Fields(t *testing.T) {
	t.Parallel()
	wc := domain.WizardConfig{Mode: "minimal", BasePath: ".", Provider: "openai"}
	assert.Equal(t, "minimal", wc.Mode)
	assert.Equal(t, ".", wc.BasePath)
	assert.Equal(t, "openai", wc.Provider)
	wc.RolesConfig = map[string]any{"k": "v"}
	assert.Len(t, wc.RolesConfig, 1)
}
