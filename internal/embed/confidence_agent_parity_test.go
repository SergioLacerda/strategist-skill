package embed_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentFlag matches `--agent <value>` in contracts, skills and roles; a value
// may list alternatives separated by `|`.
var agentFlag = regexp.MustCompile("--agent ([A-Za-z_<>|]+)")

// producerEntry matches the producer list in confidence-governance.yaml.
var producerEntry = regexp.MustCompile(`\{agent: ([a-z_]+),`)

// Every agent name the shipped contracts tell an agent to pass to
// `strategist metrics record` must be accepted by the CLI. The contracts once
// said `critic` while the CLI accepted only `response_critic`.
func TestContractAgentNamesAreAcceptedByTheRecorder(t *testing.T) {
	accepted := map[string]bool{}
	for _, agent := range telemetry.ConfidenceAgents() {
		accepted[agent] = true
	}
	seen := 0
	err := filepath.WalkDir("defaults", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !isContractText(path) {
			return walkErr
		}
		raw, err := os.ReadFile(path) //nolint:gosec // test walks the embedded defaults tree
		require.NoError(t, err)
		for _, name := range contractAgentNames(string(raw)) {
			seen++
			assert.True(t, accepted[name], "%s names --agent %q, which the recorder rejects", path, name)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Positive(t, seen, "the scan must find the documented agent names")
}

func isContractText(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".md" || ext == ".yaml"
}

func contractAgentNames(text string) []string {
	var names []string
	for _, match := range agentFlag.FindAllStringSubmatch(text, -1) {
		names = append(names, flagAlternatives(match[1])...)
	}
	for _, match := range producerEntry.FindAllStringSubmatch(text, -1) {
		names = append(names, match[1])
	}
	return names
}

// flagAlternatives splits `a|b` and drops `<placeholder>` values.
func flagAlternatives(value string) []string {
	var names []string
	for _, name := range strings.Split(value, "|") {
		if name != "" && !strings.HasPrefix(name, "<") {
			names = append(names, name)
		}
	}
	return names
}
