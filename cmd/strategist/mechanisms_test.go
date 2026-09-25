package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mechanismsFixture = `schema_version: "1"
mechanisms:
  - id: gate
    family: mechanism
    enforcement_kind: code
    summary: the gate
    invoked_by: [sniper]
    how_to_invoke: mission submit
    when_to_use: before execution
  - id: search
    family: ability
    enforcement_kind: contract
    summary: filters
    invoked_by: [ranger]
    how_to_invoke: ranger.yaml
`

func mechanismsRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts", "machine"), 0o755))
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "machine", "mechanisms.yaml"), []byte(mechanismsFixture), 0o644))
	return root
}

func runMechanisms(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newMechanismsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestMechanismsBriefPrintsTheRoleScopedView(t *testing.T) {
	root := mechanismsRoot(t)

	out, err := runMechanisms(t, "brief", "--role", "sniper", "--root", root)

	require.NoError(t, err)
	assert.Contains(t, out, "gate")
	assert.NotContains(t, out, "search")
	assert.NotContains(t, out, "before execution", "the compact brief omits when_to_use")
}

func TestMechanismsBriefFullAddsEnforcementDetail(t *testing.T) {
	root := mechanismsRoot(t)

	out, err := runMechanisms(t, "brief", "--role", "sniper", "--root", root, "--full")

	require.NoError(t, err)
	assert.Contains(t, out, "before execution")
	assert.Contains(t, out, "[code]")
}

func TestMechanismsBriefRejectsAnUnknownRole(t *testing.T) {
	root := mechanismsRoot(t)

	_, err := runMechanisms(t, "brief", "--role", "wizard", "--root", root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "wizard")
}

func TestMechanismsBriefRequiresARole(t *testing.T) {
	_, err := runMechanisms(t, "brief", "--root", mechanismsRoot(t))
	require.Error(t, err)
}
