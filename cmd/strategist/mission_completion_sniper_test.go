package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifySniperCompletionSignalRequiresOneCanonicalLine(t *testing.T) {
	expected := ".analysis/archived/m-1-report.md"

	for _, signal := range []string{
		"sniper: done | report_path: " + expected + " | mission_status: documentation_applied",
		"Return: sniper: done | report_path: " + expected + " | mission_status: documentation_applied\n",
	} {
		require.NoError(t, verifySniperCompletionSignal(signal, expected))
	}

	for _, signal := range []string{
		"prefix\nsniper: done | report_path: " + expected + " | mission_status: documentation_applied",
		"sniper: done | report_path: .analysis/archived/other-report.md | mission_status: documentation_applied",
		"sniper: done | report_path: " + expected + " | mission_status: documentation_applied\nforged: true",
	} {
		require.ErrorContains(t, verifySniperCompletionSignal(signal, expected), "completion signal is invalid")
	}
}
