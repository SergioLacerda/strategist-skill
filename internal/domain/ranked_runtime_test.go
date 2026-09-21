package domain

import (
	"os"
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

func TestValidateOpenSpecHealthcheckIsPathFormIndependent(t *testing.T) {
	base := t.TempDir()
	physical := filepath.Join(base, "real", ".strategist")
	runtimeRoot := filepath.Join(physical, "openspec")
	require.NoError(t, os.MkdirAll(runtimeRoot, 0o755))
	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(filepath.Join(base, "real"), link))
	canonical, err := filepath.EvalSymlinks(physical)
	require.NoError(t, err)
	output := []byte(`{"root":{"path":"` + filepath.ToSlash(canonical) + `"}}`)

	t.Chdir(filepath.Join(base, "real"))
	tests := []struct {
		name string
		root string
	}{
		{name: "relative", root: filepath.Join(".strategist", "openspec")},
		{name: "relative with dot segments", root: filepath.Join(".", ".strategist", "..", ".strategist", "openspec")},
		{name: "trailing slash", root: runtimeRoot + string(filepath.Separator)},
		{name: "symlinked", root: filepath.Join(link, ".strategist", "openspec")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, ValidateOpenSpecHealthcheck(output, tt.root))
		})
	}
}

func TestValidateOpenSpecHealthcheckStillRejectsOtherDirectoryWhenRootIsRelative(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, ".strategist", "openspec"), 0o755))
	other := filepath.Join(base, "other")
	require.NoError(t, os.MkdirAll(other, 0o755))
	t.Chdir(base)

	output := []byte(`{"root":{"path":"` + filepath.ToSlash(other) + `"}}`)
	err := ValidateOpenSpecHealthcheck(output, filepath.Join(".strategist", "openspec"))
	require.ErrorContains(t, err, "semantic root mismatch")
}

// Regression: drift_pipeline.txt recorded expected ".strategist" against an
// absolute observed root for a healthy runtime.
func TestValidateOpenSpecHealthcheckDriftPipelineRegression(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, ".strategist", "openspec"), 0o755))
	canonical, err := filepath.EvalSymlinks(filepath.Join(base, ".strategist"))
	require.NoError(t, err)
	t.Chdir(base)

	output := []byte(`{"root":{"path":"` + filepath.ToSlash(canonical) + `"}}`)
	require.NoError(t, ValidateOpenSpecHealthcheck(output, ".strategist/openspec"))
}

func TestOpenSpecRuntimeRootValidation(t *testing.T) {
	contract := func(root string) RankedRuntimeContract {
		return RankedRuntimeContract{Kind: RankedRuntimeOpenSpecRoot, Root: root, Bootstrap: "openspec init", Healthcheck: "openspec context --json"}
	}
	rejected := []string{
		"", ".", "..", ".strategist", ".strategist/", "openspec",
		".strategist/./openspec", ".strategist//openspec", ".strategist/openspec/",
		".strategist/../openspec", "../.strategist/openspec", ".strategist/a/../openspec",
		"/.strategist/openspec", "C:/.strategist/openspec", ".strategist/C:/openspec",
	}
	for _, root := range rejected {
		t.Run("rejects "+root, func(t *testing.T) {
			require.Error(t, contract(root).Validate())
		})
	}
	require.NoError(t, contract(".strategist/openspec").Validate())
	require.NoError(t, contract(".strategist/a/openspec").Validate())
}

// The catalog declares the root with forward slashes. On Windows the native
// separator form must be judged the same way, not rejected by a raw
// filepath.Clean comparison; on other platforms a backslash is a literal
// character, so that spelling is not a valid root.
func TestOpenSpecRuntimeRootSeparatorForms(t *testing.T) {
	contract := RankedRuntimeContract{Kind: RankedRuntimeOpenSpecRoot, Root: `.strategist\openspec`, Bootstrap: "b", Healthcheck: "h"}
	if filepath.Separator == '\\' {
		require.NoError(t, contract.Validate())
		contract.Root = `.strategist\..\openspec`
		require.Error(t, contract.Validate())
		return
	}
	require.Error(t, contract.Validate())
}

func TestRankedRuntimeExecutableMissingMessageIsActionable(t *testing.T) {
	msg := RankedRuntimeExecutableMissingMessage("openspec-propose", "openspec")

	require.Contains(t, msg, `"openspec-propose"`)
	require.Contains(t, msg, `"openspec"`)
	require.Contains(t, msg, "PATH")
	require.Contains(t, msg, "docs/runbooks/standalone-runtime-hermeticity.md")
	require.NotContains(t, msg, "exec:")
	// The binary is the usual culprit, so the message must say so and name the fix.
	require.Contains(t, msg, "built without the embedded runtime")
	require.Contains(t, msg, "make build-standalone")
	require.Contains(t, msg, "strategist version --build")
	require.NotContains(t, msg, "does not yet ship")
}

func TestRankedRuntimeContractValidatesPinnedIdentity(t *testing.T) {
	base := RankedRuntimeContract{Kind: RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec", Bootstrap: "openspec init", Healthcheck: "openspec context --json"}

	pinned := base
	pinned.Version, pinned.NodeVersion = "1.13.0", "22.23.2"
	require.NoError(t, pinned.Validate())
	require.NoError(t, base.Validate(), "identity stays optional so unpinned providers keep working")

	for _, bad := range []string{"v1.13.0", "1.13", "latest", "1.13.0 "} {
		c := base
		c.Version = bad
		require.Error(t, c.Validate(), bad)
		c = base
		c.NodeVersion = bad
		require.Error(t, c.Validate(), bad)
	}

	none := RankedRuntimeContract{Kind: RankedRuntimeNone, Version: "1.0.0"}
	require.Error(t, none.Validate(), "a provider without a runtime cannot declare a pinned version")
}

func TestParseReportedVersionAndSkew(t *testing.T) {
	for output, want := range map[string]string{
		"1.13.0\n":           "1.13.0",
		"v1.13.0":            "1.13.0",
		"  1.13.0  \r\n":     "1.13.0",
		"openspec 1.13.0":    "1.13.0",
		"":                   "",
		"no version here\n":  "",
		"1.13.0\nextra line": "1.13.0",
	} {
		require.Equal(t, want, ParseReportedVersion([]byte(output)), output)
	}

	require.False(t, VersionSkew("", "9.9.9"), "no pin, nothing to compare")
	require.False(t, VersionSkew("1.13.0", "1.13.0"))
	require.True(t, VersionSkew("1.13.0", "1.10.0"))
	require.True(t, VersionSkew("1.13.0", ""), "an unreadable version cannot confirm the pin")

	msg := RankedRuntimeVersionSkewMessage("openspec-propose", "1.13.0", "1.10.0")
	require.Contains(t, msg, ReasonRankedRuntimeVersionSkew)
	require.Contains(t, msg, "1.13.0")
	require.Contains(t, msg, "1.10.0")
}
