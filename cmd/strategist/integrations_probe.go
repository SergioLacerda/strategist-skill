package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/spf13/cobra"
)

const probeFileName = "integration-probe.json"

func probePath(root string) string { return filepath.Join(root, "memory", probeFileName) }

// runIntegrationsProbe sends one synthetic message and records the observation.
// It refuses a disabled provider and a missing credential before any call, and
// it grants no authorization: a later handoff still evaluates everything itself.
func runIntegrationsProbe(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	provider, err := probeTarget(root)
	if err != nil {
		return err
	}
	adapter, _, err := newJEVAdapter(root, provider)
	if err != nil {
		return fmt.Errorf("integrations probe: %w", err)
	}
	record := probe.Run(commandContext(cmd), provider, adapter, time.Now)
	if err := probe.Save(probePath(root), record); err != nil {
		return fmt.Errorf("integrations probe: %w", err)
	}
	if record.State != "ok" {
		if perr := printLines(cmd, "probe: failed state="+record.State); perr != nil {
			return perr
		}
		return fmt.Errorf("integrations probe: provider did not answer (%s)", record.State)
	}
	return printLines(cmd, fmt.Sprintf("probe: ok model=%s input_tokens=%d output_tokens=%d latency_ms=%d", record.Model, record.InputTokens, record.OutputTokens, record.LatencyMS))
}

// probeTarget returns the enabled provider whose credential resolves.
func probeTarget(root string) (config.Provider, error) {
	file, found, err := loadIntegrations(root)
	if err != nil {
		return config.Provider{}, err
	}
	provider, declared := file.Providers[integrationProvider]
	switch {
	case !found || !declared:
		return config.Provider{}, fmt.Errorf("integrations probe: no integration decision recorded; run `strategist integrations enable`")
	case !provider.Enabled:
		return config.Provider{}, fmt.Errorf("integrations probe: the provider is disabled")
	}
	if _, err := credential.Resolve(provider.CredentialRef, credential.WorkspaceEnv(filepath.Dir(root))); err != nil {
		state, _ := integration.StateOf(err)
		return config.Provider{}, fmt.Errorf("integrations probe: %s: the credential does not resolve", state)
	}
	return provider, nil
}
