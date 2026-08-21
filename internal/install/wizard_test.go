package install

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillYAMLExtractor returns a synthetic skill.yaml with the given active_config values.
type skillYAMLExtractor struct {
	langValues []string
	modeValues []string
}

func (s skillYAMLExtractor) Extract(_ string, _ bool) error { return nil }
func (s skillYAMLExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath != "skill.yaml" {
		return nil, fmt.Errorf("skillYAMLExtractor: not found: %s", relPath)
	}
	langs := strings.Join(s.langValues, ", ")
	modes := strings.Join(s.modeValues, ", ")
	yaml := fmt.Sprintf("active_config:\n  language:\n    values: [%s]\n  mode:\n    values: [%s]\n", langs, modes)
	return []byte(yaml), nil
}

// alwaysErrExtractor fails every ReadFile call, used to exercise loadSkillConfig's
// extractor-error fallback distinct from its YAML-parse-error fallback.
type alwaysErrExtractor struct{}

func (alwaysErrExtractor) Extract(_ string, _ bool) error { return nil }
func (alwaysErrExtractor) ReadFile(relPath string) ([]byte, error) {
	return nil, fmt.Errorf("alwaysErrExtractor: read error for %s", relPath)
}

func TestLoadSkillConfig(t *testing.T) {
	t.Parallel()

	t.Run("reads lang and mode values from skill.yaml", func(t *testing.T) {
		t.Parallel()
		cfg := loadSkillConfig(skillYAMLExtractor{
			langValues: []string{"pt", "en"},
			modeValues: []string{"pragmatic", "epic"},
		})
		assert.Equal(t, []string{"pt", "en"}, cfg.LangOptions)
		assert.Equal(t, []string{"pragmatic", "epic"}, cfg.ModeOptions)
	})

	t.Run("falls back to defaults when yaml is a valid but non-mapping scalar", func(t *testing.T) {
		t.Parallel()
		// minimalExtractor's default case echoes "skill.yaml\n" for this path — a
		// bare scalar that unmarshals successfully but can't populate the struct.
		cfg := loadSkillConfig(minimalExtractor{})
		assert.Equal(t, defaultLangOptions, cfg.LangOptions)
		assert.Equal(t, defaultModeOptions, cfg.ModeOptions)
	})

	t.Run("falls back to defaults when extractor.ReadFile errors", func(t *testing.T) {
		t.Parallel()
		cfg := loadSkillConfig(alwaysErrExtractor{})
		assert.Equal(t, defaultLangOptions, cfg.LangOptions)
		assert.Equal(t, defaultModeOptions, cfg.ModeOptions)
	})

	t.Run("falls back to defaults when yaml is malformed", func(t *testing.T) {
		t.Parallel()
		cfg := loadSkillConfig(skillYAMLExtractor{langValues: nil, modeValues: nil})
		assert.Equal(t, defaultLangOptions, cfg.LangOptions)
		assert.Equal(t, defaultModeOptions, cfg.ModeOptions)
	})
}

func TestNormLang(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pt-BR", normLang("pt"))
	assert.Equal(t, "pt-BR", normLang("pt-BR"))
	assert.Equal(t, "pt-BR", normLang("PT-BR"))
	assert.Equal(t, "en", normLang("en"))
	assert.Equal(t, "fr", normLang("fr"))
}

func TestValidateProvider(t *testing.T) {
	t.Parallel()

	assert.Empty(t, validateProvider(knownProviderRisk, "brainstorming", "write_analysis"))
	assert.Empty(t, validateProvider(knownProviderRisk, "openspec-explore", "write_analysis"))
	assert.Empty(t, validateProvider(knownProviderRisk, "sdd-ask", "controlled"))
	assert.Contains(t, validateProvider(knownProviderRisk, "brainstorming", "controlled"), "preflight will block at runtime")
	assert.Contains(t, validateProvider(knownProviderRisk, "unknown-provider", "write_analysis"), "slot plugin")
	assert.Contains(t, validateProvider(knownProviderRisk, "unknown-provider", "write_analysis"), "known plugin catalog")
}

func TestInstallableDefaultProviders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "skills/brainstorming/skill.yaml", installableDefaultProviders["brainstorming"])
	assert.Equal(t, "skills/openspec-explore/skill.yaml", installableDefaultProviders["openspec-explore"])
}

func TestRunWizard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		wantUILanguage string
		wantDocLang    string
		wantChatLang   string
		wantCodeLang   string
		wantMode       string
		wantBase       string
		wantDiscovery  string
		wantRefinement string
		wantExecution  string
		wantChestPath  string
	}{
		{
			name: "all defaults (empty lines)",
			// 10 prompts: ui/doc/chat/code/mode/base/discovery/refinement/execution/chest
			input:          "\n\n\n\n\n\n\n\n\n\n",
			wantUILanguage: "en",
			wantDocLang:    "en",
			wantChatLang:   "en",
			wantCodeLang:   "en",
			wantMode:       "epic",
			wantBase:       ".analysis",
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sniper",
			wantChestPath:  "",
		},
		{
			name:           "en ui, custom languages and slots with chest",
			input:          "en\nen\npt-BR\nen\nepic\n/workspace\nbrainstorming\narchivist\nbatata\n.sdd/source\n",
			wantUILanguage: "en",
			wantDocLang:    "en",
			wantChatLang:   "pt-BR",
			wantCodeLang:   "en",
			wantMode:       "epic",
			wantBase:       "/workspace",
			wantDiscovery:  "brainstorming",
			wantRefinement: "archivist",
			// Legacy execution input ("batata") is consumed but discarded — execution
			// always resolves to the native sniper role, never a scripted/custom value.
			wantExecution: "sniper",
			wantChestPath: ".sdd/source",
		},
		{
			name:           "pt-BR ui language",
			input:          "pt-BR\nen\npt-BR\nen\npragmatic\n.\n\n\n\n\n",
			wantUILanguage: "pt-BR",
			wantDocLang:    "en",
			wantChatLang:   "pt-BR",
			wantCodeLang:   "en",
			wantMode:       "pragmatic",
			wantBase:       ".",
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sniper",
			wantChestPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(tt.input)), minimalExtractor{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantUILanguage, wc.UILanguage)
			assert.Equal(t, tt.wantDocLang, wc.DocLanguage)
			assert.Equal(t, tt.wantChatLang, wc.ChatLanguage)
			assert.Equal(t, tt.wantCodeLang, wc.CodeLanguage)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantDiscovery, wc.DiscoveryProvider)
			assert.Equal(t, tt.wantRefinement, wc.RefinementProvider)
			assert.Equal(t, tt.wantExecution, wc.ExecutionProvider)
			assert.Equal(t, tt.wantChestPath, wc.TreasureChestPath)
		})
	}
}

func TestWizardDoesNotAskPermissionLevel(t *testing.T) {
	t.Parallel()
	// Input has no legacy execution_mode / apply_workspace / git_persistence_mode / adr tokens.
	// 10 prompts: ui/doc/chat/code/mode/base/discovery/refinement/execution/chest
	// If the wizard still prompts for execution mode or ADR, the input will be exhausted and the test errors.
	input := "en\nen\npt-BR\nen\nepic\n.analysis\nbrainstorming\nopenspec-explore\nsdd-ask\n\n"
	wc, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(input)), minimalExtractor{})
	require.NoError(t, err)
	assert.Equal(t, "epic", wc.Mode)
	assert.Equal(t, "brainstorming", wc.DiscoveryProvider)
}
