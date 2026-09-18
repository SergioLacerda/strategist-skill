package domain

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOpenSpecHealthcheckRejectsMalformedRootMatrix(t *testing.T) {
	runtimeRoot := filepath.Join(t.TempDir(), ".strategist", "openspec")
	expected := filepath.Dir(runtimeRoot)
	tests := []struct {
		name   string
		output string
	}{
		{name: "invalid json", output: "not-json"},
		{name: "missing root", output: `{"members":[]}`},
		{name: "physical root", output: `{"root":{"path":"` + runtimeRoot + `"}}`},
		{name: "nested root", output: `{"root":{"path":"` + filepath.Join(runtimeRoot, "openspec") + `"}}`},
		{name: "escape", output: `{"root":{"path":"` + filepath.Join(expected, "..") + `"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, ValidateOpenSpecHealthcheck([]byte(tt.output), runtimeRoot))
		})
	}
}

func TestValidateOpenSpecHealthcheckAcceptsSemanticContainerRoot(t *testing.T) {
	runtimeRoot := filepath.Join(t.TempDir(), ".strategist", "openspec")
	output := `{"root":{"path":"` + filepath.ToSlash(filepath.Dir(runtimeRoot)) + `"}}`
	require.NoError(t, ValidateOpenSpecHealthcheck([]byte(output), runtimeRoot))
}
