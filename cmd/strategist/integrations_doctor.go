package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
	"github.com/spf13/cobra"
)

// doctorReport separates what is declared, bound, enabled, compatible and
// available, because each is a different claim. "available" comes only from a
// recorded, valid probe; the doctor itself never calls the provider.
type doctorReport struct {
	Provider      string   `json:"provider"`
	Declared      bool     `json:"declared"`
	Bound         bool     `json:"bound"`
	Enabled       bool     `json:"enabled"`
	Compatible    bool     `json:"compatible"`
	Available     string   `json:"available"`
	ObservedAt    string   `json:"observed_at,omitempty"`
	ValidUntil    string   `json:"valid_until,omitempty"`
	PinnedModel   string   `json:"pinned_model,omitempty"`
	ResolvedModel string   `json:"resolved_model,omitempty"`
	Endpoint      string   `json:"endpoint,omitempty"`
	Credential    string   `json:"credential"`
	Findings      []string `json:"findings"`
}

func runIntegrationsDoctor(cmd *cobra.Command, opts integrationsOptions) error {
	root, err := integrationsRoot(opts)
	if err != nil {
		return err
	}
	file, found, err := loadIntegrations(root)
	if err != nil {
		return err
	}
	record, probed, err := probe.Load(probePath(root))
	if err != nil {
		return fmt.Errorf("integrations doctor: %w", err)
	}
	report := buildDoctorReport(root, file, found, record, probed, time.Now())
	if opts.JSON {
		return printDoctorJSON(cmd, report)
	}
	return printLines(cmd, doctorLines(report)...)
}

func buildDoctorReport(root string, file config.File, found bool, record probe.Record, probed bool, now time.Time) doctorReport {
	provider, declared := file.Providers[integrationProvider]
	report := doctorReport{Provider: integrationProvider, Declared: found && declared, Available: string(probe.Unknown), Credential: "not_configured", Findings: []string{}}
	if !report.Declared {
		return report
	}
	report.Enabled, report.PinnedModel, report.Endpoint = provider.Enabled, provider.Model, provider.Endpoint
	report.Bound, report.Compatible = bindingState(root, file, provider)
	availability, validUntil := probe.Evaluate(record, probed, provider, now)
	report.Available = string(availability)
	if probed {
		report.ObservedAt, report.ResolvedModel = record.At.UTC().Format(time.RFC3339), record.Model
	}
	if !validUntil.IsZero() {
		report.ValidUntil = validUntil.UTC().Format(time.RFC3339)
	}
	report.Credential, report.Findings = credentialState(root, provider)
	return report
}

// bindingState reports whether the compile-time registry resolves the binding the
// configuration implies, and whether the configuration itself is valid.
func bindingState(root string, file config.File, provider config.Provider) (bound, compatible bool) {
	adapter, _, err := newJEVAdapter(root, provider)
	if err != nil {
		return false, false
	}
	registry, err := integration.NewRegistry(adapter)
	if err != nil {
		return false, false
	}
	_, resolveErr := registry.Resolve(integration.Binding{Provider: integrationProvider, Version: integrationVersion, Model: provider.Model, Consumer: "handoff", Capability: integration.CapHandoffValidate})
	bound = resolveErr == nil
	return bound, bound && file.Validate() == nil
}

func credentialState(root string, provider config.Provider) (string, []string) {
	findings := []string{}
	if provider.CredentialRef == "" {
		return "not_configured", findings
	}
	state := "resolves"
	if !setup.Resolves(provider.CredentialRef, filepath.Dir(root)) {
		state = "pending"
		if provider.Enabled {
			findings = append(findings, string(integration.StateCredentialMissing))
		}
	}
	if path, ok := dotenvPath(provider.CredentialRef, filepath.Dir(root)); ok {
		findings = append(findings, credential.Hygiene(path, credential.GitIgnored)...)
	}
	return state, findings
}

// dotenvPath extracts the file of a dotenv reference, relative to the workspace.
func dotenvPath(ref, workspace string) (string, bool) {
	rest, ok := strings.CutPrefix(ref, "dotenv:")
	if !ok {
		return "", false
	}
	path, _, _ := strings.Cut(rest, "#")
	if !filepath.IsAbs(path) {
		path = filepath.Join(workspace, path)
	}
	return path, true
}

func doctorLines(r doctorReport) []string {
	findings := "none"
	if len(r.Findings) > 0 {
		findings = strings.Join(r.Findings, ", ")
	}
	lines := []string{
		"provider: " + r.Provider,
		fmt.Sprintf("declared: %t", r.Declared), fmt.Sprintf("bound: %t", r.Bound),
		fmt.Sprintf("enabled: %t", r.Enabled), fmt.Sprintf("compatible: %t", r.Compatible),
		"available: " + r.Available,
	}
	for _, kv := range [][2]string{{"observed_at", r.ObservedAt}, {"valid_until", r.ValidUntil}, {"pinned_model", r.PinnedModel}, {"resolved_model", r.ResolvedModel}, {"endpoint", r.Endpoint}} {
		if kv[1] != "" {
			lines = append(lines, kv[0]+": "+kv[1])
		}
	}
	return append(lines, "credential: "+r.Credential, "findings: "+findings)
}

func printDoctorJSON(cmd *cobra.Command, report doctorReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("integrations doctor: %w", err)
	}
	return printLines(cmd, string(raw))
}
