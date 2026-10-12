package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	dojoadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// calibrationDependencies composes the provider-facing concerns of the DOJO
// calibration commands: the field catalog of the handoff pre-check, the JEV
// judge through the Integration Mechanism, the configured threshold and the
// content-free mission signals.
func calibrationDependencies() dojoadapter.CalibrationDependencies {
	return dojoadapter.CalibrationDependencies{
		Reader:  handoffconsumer.Reader{},
		Judge:   calibrationJudge,
		Current: configuredThreshold,
		Profile: func(profile, boundary string) (string, string) {
			id := calibration.ResolveProfile(profile)
			return id, handoffconsumer.ProfileVersionOf(id, boundary)
		},
		Plan:        calibrationPlan,
		Signals:     missionSignals,
		Policy:      policyItems,
		PairJudge:   calibrationPairJudge,
		PairProfile: func() (string, string) { return handoffconsumer.PairProfileID, handoffconsumer.PairProfileVersion() },
		Endpoint:    calibrationEndpoint,
		LivePlan:    livePlan,
		Live:        runLiveSmoke,
	}
}

func calibrationJudge(root, profile string) (calibration.Judge, error) {
	consumer, err := handoffConsumer(root)
	if err != nil {
		return nil, err
	}
	if consumer == nil {
		return nil, errors.New("no provider integration is enabled: run `strategist integrations doctor` to see the state")
	}
	return handoffconsumer.Judge{Mechanism: consumer.Mechanism, Provider: consumer.Provider, Name: consumer.Name, Version: consumer.Version, Profile: profile}, nil
}

// configuredThreshold is the approval threshold the handoff uses now.
func configuredThreshold(root string) float64 {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil || !found {
		return config.DefaultThreshold
	}
	return file.Providers[integrationProvider].Threshold()
}

const defaultPlanMaxBytes = 65536

func calibrationPlan(root, profile, boundary string, cases []calibration.Case) (dojoadapter.Plan, error) {
	transition := handoffconsumer.TransitionForBoundary(boundary)
	allowed, maxBytes := planPolicy(root, transition)
	plan := dojoadapter.Plan{Fields: allowedCatalogFields(transition, allowed)}
	for _, c := range cases {
		text, err := handoffconsumer.ProjectFor(profile, transition, handoffconsumer.MapSource(c.Fields), allowed, maxBytes)
		if err != nil {
			plan.Denied++
			continue
		}
		plan.Bytes += len(text)
	}
	return plan, nil
}

// planPolicy is the operator's data policy when the provider is enabled, and the
// whole catalog otherwise, so a dry run is informative before the setup.
func planPolicy(root, transition string) ([]string, int) {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	provider, ok := file.Providers[integrationProvider]
	if err != nil || !found || !ok || !provider.Enabled {
		return handoffconsumer.Fields(transition), defaultPlanMaxBytes
	}
	return provider.DataPolicy.AllowedFields, provider.DataPolicy.MaxStateBytes
}

// allowedCatalogFields are the catalog fields the policy lets through, in order.
func allowedCatalogFields(transition string, allowed []string) []string {
	var out []string
	for _, field := range handoffconsumer.Fields(transition) {
		for _, name := range allowed {
			if name == field {
				out = append(out, field)
			}
		}
	}
	return out
}

// missionSignals reads, for a mission, each provider reading that answered and
// the gate outcome, with no content.
func missionSignals(root, missionID string) ([]dojo.MissionSignal, error) {
	outcome, err := telemetry.GateOutcomeFor(root, missionID)
	if err != nil {
		return nil, fmt.Errorf("gate outcome: %w", err)
	}
	ledger := handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}
	var signals []dojo.MissionSignal
	for _, transition := range []string{handoff.TransitionRangerToArchivist, handoffconsumer.TransitionArchivistToGate} {
		record, found, err := ledger.LatestRecord(missionID, transition)
		if err != nil {
			return nil, fmt.Errorf("integration ledger: %w", err)
		}
		if found {
			signals = append(signals, dojo.MissionSignal{Transition: transition, Answered: record.Answered, Approved: record.Approved, Confidence: record.Confidence, Threshold: record.Threshold, GateOutcome: outcome})
		}
	}
	return signals, nil
}

// policyItems is the integration layer: the approval policy against scripted
// answers, as DOJO check items.
func policyItems(now time.Time) []domain.DojoCheckItem {
	rows := handoffconsumer.PolicyMatrix(now)
	items := make([]domain.DojoCheckItem, len(rows))
	for i, r := range rows {
		items[i] = domain.DojoCheckItem{Label: "calibration policy: " + r.Name, Passed: r.Pass, Detail: r.Detail}
	}
	return items
}

func calibrationPairJudge(root string) (calibration.PairJudge, error) {
	consumer, err := handoffConsumer(root)
	if err != nil {
		return nil, err
	}
	if consumer == nil {
		return nil, errors.New("no provider integration is enabled: run `strategist integrations doctor` to see the state")
	}
	return handoffconsumer.PairJudge{Mechanism: consumer.Mechanism, Provider: consumer.Provider, Name: consumer.Name, Version: consumer.Version}, nil
}
