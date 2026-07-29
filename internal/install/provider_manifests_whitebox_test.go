package install

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knownProvidersExtractor serves a synthetic templates/known-providers.yaml.
type knownProvidersExtractor struct {
	yaml string
}

func (k knownProvidersExtractor) Extract(_ string, _ bool) error { return nil }
func (k knownProvidersExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == "templates/known-providers.yaml" {
		return []byte(k.yaml), nil
	}
	return nil, fmt.Errorf("not found: %s", relPath)
}

func TestLoadKnownProviders(t *testing.T) {
	t.Parallel()

	t.Run("reads valid providers yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers:\n  brainstorming: write_analysis\n  sdd-ask: controlled\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, "write_analysis", got["brainstorming"])
		assert.Equal(t, "controlled", got["sdd-ask"])
	})

	t.Run("falls back to static map when providers map is empty", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers: {}\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})

	t.Run("falls back to static map on invalid yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: ": invalid: yaml:\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})
}

// partialExtractor serves minimalExtractor files but fails for the given path.
type partialExtractor struct {
	failPath string
}

func (p partialExtractor) Extract(targetDir string, overwrite bool) error {
	return minimalExtractor{}.Extract(targetDir, overwrite)
}
func (p partialExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == p.failPath {
		return nil, fmt.Errorf("partialExtractor: injected failure for %s", relPath)
	}
	return minimalExtractor{}.ReadFile(relPath)
}

func TestWriteSelectedProviderManifests_ReadFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{
		Extractor:   partialExtractor{failPath: "skills/brainstorming/skill.yaml"},
		Compiler:    nopCompiler{},
		ShimHomeDir: t.TempDir(),
		WizardPrompter: NewTextPrompter(strings.NewReader(
			"en\nen\nen\nen\nepic\n.analysis\nyes\nbrainstorming\nopenspec-explore\nsdd-ask\n\n",
		)),
	}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "brainstorming")
}
