package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProvider(t *testing.T) {
	t.Parallel()

	assert.Empty(t, validateProvider("brainstorming", "write_pending"))
	assert.Empty(t, validateProvider("openspec-explore", "write_analysis"))
	assert.Empty(t, validateProvider("sdd-ask", "controlled"))
	assert.Contains(t, validateProvider("brainstorming", "controlled"), "preflight will block at runtime")
	assert.Contains(t, validateProvider("unknown-provider", "write_pending"), "is not in the known-providers registry")
}

func TestInstallableDefaultProviders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "providers/brainstorming/skill.yaml", installableDefaultProviders["brainstorming"])
	assert.Equal(t, "providers/openspec-explore/skill.yaml", installableDefaultProviders["openspec-explore"])
}

func TestRunWizard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		input             string
		wantUILanguage    string
		wantDocLanguage   string
		wantChatLanguage  string
		wantCodeLanguage  string
		wantMode          string
		wantBase          string
		wantAdrEnabled    bool
		wantExecutionMode string
		wantGitMode       string
		wantDiscovery     string
		wantRefinement    string
		wantExecution     string
		wantChestPath     string
	}{
		{
			name: "all defaults (empty lines)",
			// 13 prompts:
			// ui/doc/chat/code/mode/base/adr/executionMode/gitMode/discovery/refinement/execution/chest
			input:             "\n\n\n\n\n\n\n\n\n\n\n\n\n",
			wantUILanguage:    "en",
			wantDocLanguage:   "en",
			wantChatLanguage:  "en",
			wantCodeLanguage:  "en",
			wantMode:          "epic",
			wantBase:          ".analysis",
			wantAdrEnabled:    true,
			wantExecutionMode: "plan_only",
			wantGitMode:       "forbidden",
			wantDiscovery:     "brainstorming",
			wantRefinement:    "openspec-explore",
			wantExecution:     "sdd-ask",
			wantChestPath:     "",
		},
		{
			name:              "en ui, custom languages and slots with chest",
			input:             "en\nen\npt-BR\nen\nepic\n/workspace\nyes\napply_workspace\nexplicit_commit\nbrainstorming\narchivist\nsdd-ask-full\n.sdd/source\n",
			wantUILanguage:    "en",
			wantDocLanguage:   "en",
			wantChatLanguage:  "pt-BR",
			wantCodeLanguage:  "en",
			wantMode:          "epic",
			wantBase:          "/workspace",
			wantAdrEnabled:    true,
			wantExecutionMode: "apply_workspace",
			wantGitMode:       "explicit_commit",
			wantDiscovery:     "brainstorming",
			wantRefinement:    "archivist",
			wantExecution:     "sdd-ask-full",
			wantChestPath:     ".sdd/source",
		},
		{
			name:              "pt-BR ui language, ADR disabled",
			input:             "pt-BR\nen\npt-BR\nen\npragmatic\n.\nno\nplan_only\n\n\n\n\n\n",
			wantUILanguage:    "pt-BR",
			wantDocLanguage:   "en",
			wantChatLanguage:  "pt-BR",
			wantCodeLanguage:  "en",
			wantMode:          "pragmatic",
			wantBase:          ".",
			wantAdrEnabled:    false,
			wantExecutionMode: "plan_only",
			wantGitMode:       "forbidden",
			wantDiscovery:     "brainstorming",
			wantRefinement:    "openspec-explore",
			wantExecution:     "sdd-ask",
			wantChestPath:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(NewTextPrompter(strings.NewReader(tt.input)))
			require.NoError(t, err)
			assert.Equal(t, tt.wantUILanguage, wc.UILanguage)
			assert.Equal(t, tt.wantDocLanguage, wc.DocLanguage)
			assert.Equal(t, tt.wantChatLanguage, wc.ChatLanguage)
			assert.Equal(t, tt.wantCodeLanguage, wc.CodeLanguage)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantAdrEnabled, wc.AdrEnabled)
			assert.Equal(t, tt.wantExecutionMode, wc.ExecutionMode)
			assert.Equal(t, tt.wantGitMode, wc.GitPersistenceMode)
			assert.Equal(t, tt.wantDiscovery, wc.DiscoveryProvider)
			assert.Equal(t, tt.wantRefinement, wc.RefinementProvider)
			assert.Equal(t, tt.wantExecution, wc.ExecutionProvider)
			assert.Equal(t, tt.wantChestPath, wc.TreasureChestPath)
		})
	}
}
