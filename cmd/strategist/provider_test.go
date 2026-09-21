package main

import (
	"bytes"
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
