package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWizard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		input            string
		wantUILanguage   string
		wantDocLanguage  string
		wantChatLanguage string
		wantCodeLanguage string
		wantMode         string
		wantBase         string
		wantAdrEnabled   bool
		wantDoneScope    string
		wantApplyChanges bool
		wantMissionMode  string
		wantDiscovery    string
		wantRefinement   string
		wantExecution    string
		wantChestPath    string
	}{
		{
			name: "all defaults (empty lines)",
			// 12 prompts:
			// ui/doc/chat/code/mode/base/adr/missionMode/discovery/refinement/execution/chest
			input:            "\n\n\n\n\n\n\n\n\n\n\n\n",
			wantUILanguage:   "en",
			wantDocLanguage:  "en",
			wantChatLanguage: "en",
			wantCodeLanguage: "en",
			wantMode:         "epic",
			wantBase:         ".analysis",
			wantAdrEnabled:   true,
			wantMissionMode:  "entrega_executada",
			wantDoneScope:    "entrega",
			wantApplyChanges: true,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "openspec-explore",
			wantExecution:    "sdd-ask",
			wantChestPath:    "",
		},
		{
			name:             "en ui, custom languages and slots with chest",
			input:            "en\nen\npt-BR\nen\nepic\n/workspace\nyes\nentrega_revisada\nbrainstorming\narchivist\nsdd-ask-full\n.sdd/source\n",
			wantUILanguage:   "en",
			wantDocLanguage:  "en",
			wantChatLanguage: "pt-BR",
			wantCodeLanguage: "en",
			wantMode:         "epic",
			wantBase:         "/workspace",
			wantAdrEnabled:   true,
			wantMissionMode:  "entrega_revisada",
			wantDoneScope:    "entrega",
			wantApplyChanges: false,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "archivist",
			wantExecution:    "sdd-ask-full",
			wantChestPath:    ".sdd/source",
		},
		{
			name:             "pt-BR ui language, ADR disabled",
			input:            "pt-BR\nen\npt-BR\nen\npragmatic\n.\nno\nanalise\n\n\n\n\n",
			wantUILanguage:   "pt-BR",
			wantDocLanguage:  "en",
			wantChatLanguage: "pt-BR",
			wantCodeLanguage: "en",
			wantMode:         "pragmatic",
			wantBase:         ".",
			wantAdrEnabled:   false,
			wantMissionMode:  "analise",
			wantDoneScope:    "analise",
			wantApplyChanges: false,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "openspec-explore",
			wantExecution:    "sdd-ask",
			wantChestPath:    "",
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
			assert.Equal(t, tt.wantMissionMode, wc.MissionMode)
			assert.Equal(t, tt.wantDoneScope, wc.DoneScope)
			assert.Equal(t, tt.wantApplyChanges, wc.ApplyChanges)
			assert.Equal(t, tt.wantDiscovery, wc.DiscoveryProvider)
			assert.Equal(t, tt.wantRefinement, wc.RefinementProvider)
			assert.Equal(t, tt.wantExecution, wc.ExecutionProvider)
			assert.Equal(t, tt.wantChestPath, wc.TreasureChestPath)
		})
	}
}
