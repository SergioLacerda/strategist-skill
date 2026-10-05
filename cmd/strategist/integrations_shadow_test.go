package main

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shadowOn(t *testing.T, root string) bool {
	t.Helper()
	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	return file.Providers["jev"].Shadow
}

func TestShadowOnAndOffChangeOnlyTheShadowFlagAndTheDoctorSaysSo(t *testing.T) {
	root := calibrationRig(t)
	require.False(t, shadowOn(t, root))

	out, err := runIntegrations(t, func(c *cobraCmd, o integrationsOptions) error { return runShadowSet(c, o, true) }, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "shadow: on")
	assert.True(t, shadowOn(t, root))
	doctor, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, doctor, "shadow: on (readings are recorded, none decides)")

	out, err = runIntegrations(t, func(c *cobraCmd, o integrationsOptions) error { return runShadowSet(c, o, false) }, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "shadow: off")
	assert.False(t, shadowOn(t, root))
	doctor, err = runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, doctor, "shadow: off")
}

func TestShadowNeedsADeclaredProvider(t *testing.T) {
	_, err := runIntegrations(t, func(c *cobraCmd, o integrationsOptions) error { return runShadowSet(c, o, true) }, integrationsOptions{Root: bareRoot(t)})
	require.ErrorContains(t, err, "no provider is declared")
}

func TestShadowReportSummarizesOnlyShadowReadings(t *testing.T) {
	root := calibrationRig(t)
	out, err := runIntegrations(t, runShadowReport, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "no shadow readings recorded")

	ledger := handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}
	for _, record := range []handoffconsumer.Record{
		{Transition: handoffconsumer.TransitionArchivistToGate, MissionID: "m1", Shadow: true, Answered: true, Approved: true, Confidence: 0.9},
		{Transition: handoffconsumer.TransitionArchivistToGate, MissionID: "m2", Shadow: true, Answered: true, Confidence: 0.3, Hints: []string{"validation_stated"}},
		{Transition: handoffconsumer.TransitionArchivistToGate, MissionID: "m3", Answered: true, Approved: true, Confidence: 0.95},
	} {
		require.NoError(t, ledger.Append(record))
	}

	out, err = runIntegrations(t, runShadowReport, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "archivist_to_gate: 2 readings over 2 missions, 2 answered, would approve 1")
	assert.Contains(t, out, "criteria found unmet: validation_stated (1)")
	assert.Contains(t, out, "strategist dojo candidate --mission")
}
