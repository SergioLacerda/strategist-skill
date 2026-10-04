package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderOutputFormatsAndWriteFailures(t *testing.T) {
	value := map[string]string{"provider": "x"}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	require.NoError(t, printProviderOutput(cmd, value, "json"))
	require.NoError(t, printProviderOutput(cmd, value, "yaml"))
	assert.Contains(t, out.String(), "provider")
	require.ErrorContains(t, printProviderOutput(cmd, value, "xml"), "unsupported --format")

	broken := &cobra.Command{}
	broken.SetOut(failingWriter{})
	require.ErrorContains(t, printProviderJSON(broken, value), "encode provider output")
	require.ErrorContains(t, printProviderYAML(broken, value), "write provider output")
	require.ErrorContains(t, writeProviderLine(failingWriter{}, "x=%s\n", "y"), "write provider output")
}

func TestProviderCommandsRejectBadInput(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.ErrorContains(t, runProviderAdd(cmd, []string{"src"}, "", "", providerOutputOptions{}), "--slot is required")

	t.Chdir(t.TempDir())
	require.ErrorContains(t, runProviderAdd(cmd, []string{"src"}, "", "discovery", providerOutputOptions{}), "provider add:")

	_, root := workspaceWithRoot(t)
	err := runProviderAdd(cmd, []string{filepath.Join(t.TempDir(), "absent")}, root, "discovery", providerOutputOptions{Format: "json"})
	require.ErrorContains(t, err, "provider add:")

	err = runProviderValidate(cmd, []string{filepath.Join(t.TempDir(), "absent")}, providerOutputOptions{Format: "json"})
	require.ErrorContains(t, err, "provider validate:")
	require.ErrorContains(t, runProviderValidate(cmd, []string{"x"}, providerOutputOptions{Format: "xml"}), "unsupported --format")
}

func TestReadMissionCompletionRejectsMalformedInput(t *testing.T) {
	read := func(input string) error {
		cmd := &cobra.Command{}
		cmd.SetIn(strings.NewReader(input))
		_, err := readMissionCompletion(cmd)
		return err
	}
	require.ErrorContains(t, read("{nope"), "read completion JSON")
	require.ErrorContains(t, read(`{"request_id":"a"} {"request_id":"b"}`), "more than one object")
	require.ErrorContains(t, read(`{"request_id":"a"} {broken`), "trailing data")
	require.NoError(t, read(`{"request_id":"a"}`))
}
