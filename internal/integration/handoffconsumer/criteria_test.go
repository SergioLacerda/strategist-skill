package handoffconsumer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var transitions = []string{handoff.TransitionRangerToArchivist, handoff.TransitionArchivistToSniper}

func schemaFields(t *testing.T, transition string) map[string]any {
	t.Helper()
	name := map[string]string{
		handoff.TransitionRangerToArchivist: "handoff-ranger-to-archivist.schema.yaml",
		handoff.TransitionArchivistToSniper: "handoff-archivist-to-sniper.schema.yaml",
	}[transition]
	raw, err := os.ReadFile(filepath.Join("..", "..", "embed", "defaults", "schemas", name))
	require.NoError(t, err)
	var doc struct {
		Required map[string]any `yaml:"required_fields"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	return doc.Required
}

func TestEveryCriterionMapsToAnExistingContractField(t *testing.T) {
	for _, transition := range transitions {
		t.Run(transition, func(t *testing.T) {
			fields := schemaFields(t, transition)
			criteria := Criteria(transition)
			require.NotEmpty(t, criteria)
			seen := map[string]bool{}
			for _, criterion := range criteria {
				require.Contains(t, fields, criterion.Field, "criterion %s names a field the handoff schema does not declare", criterion.Key)
				require.False(t, seen[criterion.Key], "duplicate criterion key %s", criterion.Key)
				require.NotEqual(t, AggregateKey, criterion.Key)
				require.NotEmpty(t, criterion.Instructions)
				seen[criterion.Key] = true
			}
		})
	}
}

func TestQuestionsCarryTheAggregateChoiceAndOneNoulPerCriterion(t *testing.T) {
	for _, transition := range transitions {
		questions := Questions(transition)
		require.Len(t, questions, len(Criteria(transition))+1)

		aggregate := questions[0]
		require.Equal(t, AggregateKey, aggregate.Key)
		require.Equal(t, integration.KindChoice, aggregate.Kind, "only a choice answer carries a confidence criterion")
		require.Equal(t, []string{OptionAccept, OptionReject}, []string{aggregate.Options[0].Key, aggregate.Options[1].Key})
		for _, criterion := range Criteria(transition) {
			require.Contains(t, aggregate.Options[0].Description, criterion.Instructions, "the aggregate states every criterion")
		}
		for _, question := range questions[1:] {
			require.Equal(t, integration.KindNoul, question.Kind)
		}
	}
}

func TestUnknownTransitionHasNoCriteria(t *testing.T) {
	require.Empty(t, Criteria("sniper_to_validation"))
	require.Empty(t, Questions("sniper_to_validation"))
	require.Empty(t, Fields("sniper_to_validation"))
}
