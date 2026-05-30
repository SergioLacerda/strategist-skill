//go:build integration

package tests_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fixture represents a single YAML test fixture file.
type fixture struct {
	Scenario      string `yaml:"scenario"`
	Description   string `yaml:"description"`
	ExpectedEvent string `yaml:"expected_event"`
}

// expectedEventPattern matches [Strategist] phase=<word> status=<word> ...
// The fixture only specifies key=value fragments; we validate the format is parseable.
var validEventFragment = regexp.MustCompile(`^\w+=\S+`)

func TestFixtures_ValidFormat(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob(filepath.Join("fixtures", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no fixtures found in tests/fixtures/")

	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(f)
			require.NoError(t, err)

			var fix fixture
			require.NoError(t, yaml.Unmarshal(data, &fix), "fixture must be valid YAML")

			assert.NotEmpty(t, fix.Scenario, "fixture must have a scenario field")
			assert.NotEmpty(t, fix.ExpectedEvent, "fixture must have an expected_event field")

			// Validate that every token in expected_event matches key=value format
			for _, token := range strings.Fields(fix.ExpectedEvent) {
				assert.True(t, validEventFragment.MatchString(token),
					"expected_event token %q must match key=value", token)
			}
		})
	}
}
