package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	providerpkg "github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRunProviderValidateJSON(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	var output bytes.Buffer
	cmd.SetOut(&output)
	err := runProviderValidate(cmd, []string{"../../internal/embed/defaults/plugins/fixtures/minimal-provider"}, providerOutputOptions{Format: "json"})
	require.NoError(t, err)
	require.Contains(t, output.String(), `"provider_id":"fixture-provider"`)
	require.Contains(t, output.String(), `"live_invocation"`)
}

func TestPrintProviderHumanReportsUnknownLiveEvidence(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	var output bytes.Buffer
	cmd.SetOut(&output)
	report := providerpkg.Report{ProviderID: "fixture-provider", Version: "1.0.0", PackageDigest: "sha256:test", Validated: true, LiveInvocation: domain.ReadinessCheck{Status: domain.ReadinessUnknown}}
	require.NoError(t, printProviderOutput(cmd, report, "table"))
	require.Contains(t, output.String(), "live_invocation=unknown")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func sampleAddResult() providerpkg.AddResult {
	return providerpkg.AddResult{
		Report:            providerpkg.Report{ProviderID: "fixture-provider", RequestedSlot: "discovery", LiveInvocation: domain.ReadinessCheck{Status: domain.ReadinessUnknown}},
		InstanceID:        "fixture-provider@1.0.0",
		BindingGeneration: 2,
		TransactionState:  "complete",
	}
}

func TestPrintProviderOutputFormatsAnAddResult(t *testing.T) {
	for format, want := range map[string]string{
		"table": "instance=fixture-provider@1.0.0",
		"":      "generation=2",
		"json":  `"instance_id":"fixture-provider@1.0.0"`,
		"yaml":  "instance_id: fixture-provider@1.0.0",
	} {
		cmd := &cobra.Command{Use: "provider"}
		var output bytes.Buffer
		cmd.SetOut(&output)

		require.NoError(t, printProviderOutput(cmd, sampleAddResult(), format), format)
		require.Contains(t, output.String(), want, format)
	}
}

func TestPrintProviderOutputRejectsUnsupportedFormatAndType(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	cmd.SetOut(&bytes.Buffer{})

	require.ErrorContains(t, printProviderOutput(cmd, sampleAddResult(), "xml"), `unsupported --format "xml"`)
	require.ErrorContains(t, printProviderOutput(cmd, 42, "table"), "unsupported output type int")
}

func TestPrintProviderHumanListsEveryReasonAndTheInvalidStatus(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	var output bytes.Buffer
	cmd.SetOut(&output)
	report := providerpkg.Report{ProviderID: "p", Validated: false, Reasons: []providerpkg.Reason{{Code: "manifest_missing", Detail: "package.yaml"}, {Code: "slot_invalid", Detail: "x"}}}

	require.NoError(t, printProviderOutput(cmd, report, "table"))

	require.Contains(t, output.String(), "status=invalid")
	require.Contains(t, output.String(), "reason=manifest_missing detail=package.yaml")
	require.Contains(t, output.String(), "reason=slot_invalid detail=x")
}

func TestPrintProviderOutputReportsAFailedWrite(t *testing.T) {
	for _, format := range []string{"json", "yaml", "table"} {
		cmd := &cobra.Command{Use: "provider"}
		cmd.SetOut(failingWriter{})

		err := printProviderOutput(cmd, sampleAddResult(), format)

		require.ErrorContains(t, err, "disk full", format)
	}
	cmd := &cobra.Command{Use: "provider"}
	cmd.SetOut(failingWriter{})
	require.ErrorContains(t, printProviderOutput(cmd, providerpkg.Report{ProviderID: "p", Reasons: []providerpkg.Reason{{Code: "c", Detail: "d"}}}, "table"), "disk full")
}

func TestRunProviderAddRequiresASlotAndReportsAFailedAdd(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	var output bytes.Buffer
	cmd.SetOut(&output)
	fixture := "../../internal/embed/defaults/plugins/fixtures/minimal-provider"

	require.ErrorContains(t, runProviderAdd(cmd, []string{fixture}, t.TempDir(), "", providerOutputOptions{Format: "json"}), "--slot is required")

	err := runProviderAdd(cmd, []string{"/definitely/not/a/provider"}, t.TempDir(), "discovery", providerOutputOptions{Format: "json"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider")
}

func TestRunProviderValidateReportsAnInvalidSourceAfterPrintingTheReport(t *testing.T) {
	cmd := &cobra.Command{Use: "provider"}
	var output bytes.Buffer
	cmd.SetOut(&output)

	err := runProviderValidate(cmd, []string{"/definitely/not/a/provider"}, providerOutputOptions{Format: "table"})

	require.ErrorContains(t, err, "provider validate")
	require.Contains(t, output.String(), "status=invalid", "the report is printed before the error is returned")
}
