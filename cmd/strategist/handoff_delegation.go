package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	"github.com/SergioLacerda/strategist-skill/internal/integration/runbookconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

const (
	integrationProvider = "jev"
	integrationVersion  = "1"
	ledgerFileName      = "integration-calls.jsonl"
)

// delegationRoundTripper is a test seam for the provider transport; nil in
// production, where the transport builds its own HTTPS client.
var delegationRoundTripper http.RoundTripper

// handoffConsumer builds the HANDOFF pre-check consumer from the operator's
// integration file. It returns nil when no integration is enabled, which is the
// normal case: the handoff then runs on main alone.
func handoffConsumer(root string) (*handoffconsumer.Consumer, error) {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		return nil, fmt.Errorf("integration configuration: %w", err)
	}
	provider, ok := file.Providers[integrationProvider]
	if !found || !ok || !provider.Enabled {
		return nil, nil
	}
	adapter, env, err := newJEVAdapter(root, provider)
	if err != nil {
		return nil, err
	}
	registry, err := integration.NewRegistry(adapter)
	if err != nil {
		return nil, fmt.Errorf("integration registry: %w", err)
	}
	return &handoffconsumer.Consumer{
		Mechanism: mechanism.New(mechanism.Options{
			Config: file, Registry: registry, Env: env, Breaker: policy.NewBreaker(3, time.Minute, time.Now),
		}),
		Provider: provider, Name: integrationProvider, Version: integrationVersion,
		Ledger: handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}, Now: time.Now,
		Calibrations: calibrationResolver(root),
	}, nil
}

func runbookConsumer(root string) (*runbookconsumer.Consumer, error) {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		return nil, fmt.Errorf("integration configuration: %w", err)
	}
	provider, ok := file.Providers[integrationProvider]
	if !found || !ok || !provider.Enabled {
		return nil, nil
	}
	adapter, env, err := newJEVAdapter(root, provider)
	if err != nil {
		return nil, err
	}
	registry, err := integration.NewRegistry(adapter)
	if err != nil {
		return nil, fmt.Errorf("integration registry: %w", err)
	}
	return &runbookconsumer.Consumer{
		Mechanism: mechanism.New(mechanism.Options{Config: file, Registry: registry, Env: env, Breaker: policy.NewBreaker(3, time.Minute, time.Now)}),
		Provider:  provider, Name: integrationProvider, Version: integrationVersion,
		Ledger: handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}, Now: time.Now,
	}, nil
}

// newJEVAdapter builds the JEV adapter and the credential environment for the
// operator's configuration. The secret is resolved per call, never here.
func newJEVAdapter(root string, provider config.Provider) (*jev.Adapter, credential.Env, error) {
	client, err := transport.New(transport.Config{Endpoint: provider.Endpoint, RoundTripper: delegationRoundTripper})
	if err != nil {
		return nil, credential.Env{}, fmt.Errorf("integration transport: %w", err)
	}
	env := credential.WorkspaceEnv(filepath.Dir(root))
	secret := func() (credential.Secret, error) { return credential.Resolve(provider.CredentialRef, env) }
	adapter, err := jev.New(client, secret, jev.Config{Model: provider.Model, AllowAlias: provider.AllowModelAlias})
	if err != nil {
		return nil, credential.Env{}, fmt.Errorf("integration adapter: %w", err)
	}
	return adapter, env, nil
}

// precheckHandoff runs one pre-check. Nothing here can fail a handoff: a
// configuration or construction problem is reported as a fallback and main
// proceeds.
func precheckHandoff(ctx context.Context, root, missionID, transition, subject string, source handoffconsumer.Source) handoffconsumer.Report {
	consumer, err := handoffConsumer(root)
	if err != nil {
		return handoffconsumer.Report{Status: handoffconsumer.StatusFallback, Reason: integration.StateIncompatibleContract}
	}
	if consumer == nil {
		return handoffconsumer.Report{Status: handoffconsumer.StatusDisabled, Reason: integration.StateDisabled}
	}
	request := handoffconsumer.Request{MissionID: missionID, Transition: transition, Subject: subject, Source: source}
	report, reused := consumer.Reuse(request)
	if !reused {
		report = consumer.Precheck(ctx, request)
	}
	recordProviderConfidence(root, request, report)
	return report
}

// recordProviderConfidence reflects the reading in the confidence ledger. A
// failure is logged and never blocks the handoff: the provider is advisory.
func recordProviderConfidence(root string, request handoffconsumer.Request, report handoffconsumer.Report) {
	if err := handoffconsumer.RecordConfidence(telemetry.ConfidenceHistoryPath(root), request, report, "handoff-"+request.Transition); err != nil {
		slog.Warn("[Strategist] provider confidence was not recorded", telemetry.AttrComponent, "handoff", "transition", request.Transition, "reason", err.Error())
	}
}
