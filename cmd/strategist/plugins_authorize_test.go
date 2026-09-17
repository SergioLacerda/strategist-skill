package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/authorization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginsAuthorizeCmd_IsRegistered(t *testing.T) {
	var found bool
	for _, command := range pluginsCmd.Commands() {
		if command.Use == "authorize" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected authorize to be registered under plugins")
}

func TestExitCodeForAuthorizationStates(t *testing.T) {
	assert.Equal(t, 2, exitCodeFor(fmt.Errorf("%w: denied", authorization.ErrDenied)))
}

func TestRunPluginsAuthorizeRequiresTarget(t *testing.T) {
	err := runPluginsAuthorize(pluginsAuthorizeOptions{})
	require.EqualError(t, err, "plugins authorize: --target is required")
}

func TestResolvePluginsAuthorizeRootUsesConfiguredRoot(t *testing.T) {
	configured := filepath.Join(t.TempDir(), ".strategist")
	got, err := resolvePluginsAuthorizeRoot(configured)
	require.NoError(t, err)
	want, err := filepath.Abs(configured)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestRunPluginsAuthorizePrintsBlockedReport(t *testing.T) {
	root := t.TempDir()
	out := captureStdout(t, func() {
		err := runPluginsAuthorize(pluginsAuthorizeOptions{
			Root: root, Target: "docs/next.md", JSON: true,
		})
		require.ErrorIs(t, err, authorization.ErrBlocked)
	})
	assert.Contains(t, out, `"decision": "blocked"`)
	assert.Contains(t, out, `"reason_code": "runtime_unavailable"`)
}

func TestPrintPluginsAuthorizationTable(t *testing.T) {
	out := captureStdout(t, func() {
		err := printPluginsAuthorization(pluginsAuthorizeOptions{}, authorization.Report{
			Decision: "allowed", ReasonCode: "ok", Target: "docs/readme.md", Permission: "documentation",
			Dimensions: []authorization.Dimension{{Name: "runtime", Status: "ready", ReasonCode: "verified", EvidenceState: "static"}},
		})
		require.NoError(t, err)
	})
	assert.Contains(t, out, "authorization=allowed reason=ok target=docs/readme.md permission=documentation")
	assert.Contains(t, out, "runtime=ready reason=verified evidence=static")
}

func TestPrintPluginsAuthorizationJSON(t *testing.T) {
	out := captureStdout(t, func() {
		err := printPluginsAuthorization(pluginsAuthorizeOptions{JSON: true}, authorization.Report{
			SchemaVersion: authorization.ReportSchemaVersion, Decision: "denied", ReasonCode: "blocked",
			Target: "src/main.go", Dimensions: []authorization.Dimension{},
		})
		require.NoError(t, err)
	})
	assert.Contains(t, out, `"schema_version": "strategist-authorization-report/v1"`)
	assert.Contains(t, out, `"target": "src/main.go"`)
}
