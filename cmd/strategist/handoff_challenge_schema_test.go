package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The --challenges/--ack shapes are documented in
// schemas/handoff-challenge.schema.yaml so agents do not have to read Go
// source; this test fails when the documented fields and the decoded DTO
// fields diverge.
func TestHandoffChallengeSchemaMatchesTheDecodedFiles(t *testing.T) {
	raw, err := embedpkg.Extractor{}.ReadFile("schemas/handoff-challenge.schema.yaml")
	require.NoError(t, err)
	var schema struct {
		Challenges struct {
			ItemFields map[string]any `yaml:"item_fields"`
		} `yaml:"challenges_file"`
		Ack struct {
			Fields map[string]any `yaml:"fields"`
		} `yaml:"ack_file"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &schema))

	require.Equal(t, yamlTags(reflect.TypeOf(handoffChallengeYAML{})), sortedKeys(schema.Challenges.ItemFields))
	require.Equal(t, yamlTags(reflect.TypeOf(handoffAckYAML{})), sortedKeys(schema.Ack.Fields))
}

func yamlTags(typ reflect.Type) []string {
	var tags []string
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			tags = append(tags, name)
		}
	}
	sort.Strings(tags)
	return tags
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
