package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every command that reads a JSON/YAML payload names its shape or schema in
// --help, so an agent never has to read Go source to build the payload.
func TestPayloadCommandsDescribeTheirInputInHelp(t *testing.T) {
	cases := map[string]struct {
		path []string
		want []string
	}{
		"mission route":           {[]string{"mission", "route"}, []string{"stdin", "schemas/scout-route-decision.schema.yaml"}},
		"mission complete":        {[]string{"mission", "complete"}, []string{"stdin", `"request_id"`, `"result"`}},
		"metrics record":          {[]string{"metrics", "record"}, []string{"--claim-file -", "claim:", "claims:", "evidence:"}},
		"handoff evaluate-ranger": {[]string{"handoff", "evaluate-ranger"}, []string{"schemas/handoff-challenge.schema.yaml"}},
		"handoff evaluate":        {[]string{"handoff", "evaluate"}, []string{"schemas/handoff-challenge.schema.yaml"}},
		"handoff verify":          {[]string{"handoff", "verify"}, []string{"schemas/handoff-challenge.schema.yaml"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(tc.path)
			require.NoError(t, err)
			require.Equal(t, tc.path[len(tc.path)-1], cmd.Name())
			for _, want := range tc.want {
				assert.Contains(t, cmd.Long, want)
			}
			assert.Contains(t, cmd.Long, "byte order mark", "the encoding rule is stated")
		})
	}
}
