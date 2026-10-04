package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
	"github.com/spf13/cobra"
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
	return consumer.Precheck(ctx, handoffconsumer.Request{MissionID: missionID, Transition: transition, Subject: subject, Source: source})
}

// rangerPrecheck checks the Ranger analysis artifact of a mission.
func rangerPrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	artifact := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	subject, err := handoff.RangerArtifactDigest(artifact)
	if err != nil {
		return unreadableReport()
	}
	source, err := handoffconsumer.NewRangerSource(artifact)
	if err != nil {
		return unreadableReport()
	}
	return precheckHandoff(ctx, root, missionID, handoff.TransitionRangerToArchivist, subject, source)
}

// archivistPrecheck checks the refined package of a mission.
func archivistPrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	refined := filepath.Join(basePath, "refined", missionID)
	subject, err := handoff.PackageDigest(refined)
	if err != nil {
		return unreadableReport()
	}
	source, err := handoffconsumer.NewPackageSource(refined)
	if err != nil {
		return unreadableReport()
	}
	return precheckHandoff(ctx, root, missionID, handoff.TransitionArchivistToSniper, subject, source)
}

// unreadableReport is the silent case: the artifact cannot be read here, and the
// main path will report that precisely, so the pre-check simply does not apply.
func unreadableReport() handoffconsumer.Report {
	return handoffconsumer.Report{Status: handoffconsumer.StatusDisabled, Reason: integration.StateDisabled}
}

// rangerDelegate is the callback for the automatic Ranger skip: it pre-checks
// the artifact only when the skip is about to be recorded.
func rangerDelegate(ctx context.Context, root, basePath, missionID string) func(subject string) *handoff.Delegation {
	return func(string) *handoff.Delegation {
		return rangerPrecheck(ctx, root, basePath, missionID).Delegation
	}
}

// printDelegation tells the agent what the pre-check decided. It prints nothing
// when no integration is enabled, so existing output is unchanged.
func printDelegation(cmd *cobra.Command, report handoffconsumer.Report) {
	if report.Status == handoffconsumer.StatusDisabled {
		return
	}
	var line strings.Builder
	fmt.Fprintf(&line, "delegation: %s", report.Status)
	switch report.Status { //nolint:exhaustive // approved and signal print their detail; every other status prints its reason
	case handoffconsumer.StatusApproved:
		fmt.Fprintf(&line, " provider=%s model=%s confidence=%.2f threshold=%.2f", report.Delegation.Provider, report.Delegation.Model, report.Confidence, report.Threshold)
	case handoffconsumer.StatusSignal:
		fmt.Fprintf(&line, " confidence=%.2f threshold=%.2f path=main hints=%s", report.Confidence, report.Threshold, hintList(report.Hints))
	default:
		fmt.Fprintf(&line, " reason=%s path=main", report.Reason)
	}
	if report.Decision.Suspect {
		line.WriteString(" integrity=suspect")
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), line.String()); err != nil {
		return
	}
}

// commandContext returns the command's context, or a background one when the
// command was built without it (tests, direct calls).
func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func hintList(hints []handoffconsumer.Hint) string {
	names := make([]string, 0, len(hints))
	for _, hint := range hints {
		names = append(names, hint.Criterion)
	}
	return strings.Join(names, ",")
}
