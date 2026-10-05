package main

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/spf13/cobra"
)

func runCalibrationList(cmd *cobra.Command, opts integrationsOptions) error {
	_, basePath, err := calibrationBase(opts)
	if err != nil {
		return err
	}
	ids, err := dojo.ListCalibrations(basePath)
	if err != nil {
		return fmt.Errorf("integrations calibration: %w", err)
	}
	lines := []string{}
	for _, id := range ids {
		record, loadErr := dojo.LoadCalibration(basePath, id)
		if loadErr != nil {
			lines = append(lines, id+"\tunreadable")
			continue
		}
		facts := handoffconsumer.FactsOf(record)
		lines = append(lines, fmt.Sprintf("%s\t%s\tmodel=%s\tprofile=%s\t%s", id, facts.Status, facts.Model, facts.Profile, facts.Date))
	}
	if len(lines) == 0 {
		lines = append(lines, "no calibration published yet: run `strategist dojo calibrate <case>`")
	}
	return printLines(cmd, lines...)
}

func runCalibrationShow(cmd *cobra.Command, opts integrationsOptions, id string) error {
	root, basePath, err := calibrationBase(opts)
	if err != nil {
		return err
	}
	if id == "" {
		file, _, loadErr := loadIntegrations(root)
		if loadErr != nil {
			return loadErr
		}
		if id = file.Providers[integrationProvider].CalibrationID; id == "" {
			return printLines(cmd, "no calibration is pinned: the configured threshold applies (não calibrado)")
		}
	}
	record, err := dojo.LoadCalibration(basePath, id)
	if err != nil {
		return fmt.Errorf("integrations calibration: %w", err)
	}
	return printLines(cmd, calibrationLines(record)...)
}

func calibrationLines(record calibration.Record) []string {
	facts := handoffconsumer.FactsOf(record)
	lines := []string{
		"calibration: " + record.ID, "status: " + string(record.Status),
		fmt.Sprintf("bound to: %s %s, model %s, profile %s (%s)", record.Provider, facts.Date, record.Model, record.Profile, record.ProfileVersion),
		fmt.Sprintf("risk policy: false approval <= %.1f%%, false rejection <= %.1f%%", 100*record.Policy.MaxFalseApproval, 100*record.Policy.MaxFalseRejection),
		fmt.Sprintf("sample: %d golden positives, %d defect negatives, corpus %s, catalog %s, seed %d", facts.Golden, facts.Defects, record.CorpusDigest, record.CatalogVersion, record.Seed),
		fmt.Sprintf("judge: auc %.3f, ece %.3f, brier %.3f", record.AUC, record.ECE, record.Brier),
	}
	for _, gate := range record.Gates {
		lines = append(lines, fmt.Sprintf("gate %s: %s (%.3f, limit %.3f)", gate.Name, gate.Status, gate.Value, gate.Limit))
	}
	if record.Threshold != nil && record.HeldOut != nil {
		lines = append(lines, fmt.Sprintf("recommended threshold: %.2f (held-out false approval %.1f%% [%.1f-%.1f], false rejection %.1f%% on %d defects and %d positives)",
			*record.Threshold, 100*record.HeldOut.FalseApproval.Value, 100*record.HeldOut.FalseApproval.Low, 100*record.HeldOut.FalseApproval.High, 100*record.HeldOut.FalseRejection.Value, record.HeldOut.Defects, record.HeldOut.Positives))
	}
	if record.FailingGate != "" || len(record.Notes) > 0 {
		lines = append(lines, fmt.Sprintf("no recommendation: failing gate %q, notes %v", record.FailingGate, record.Notes))
	}
	return append(lines, fmt.Sprintf("current threshold would approve %.1f%% of defects and reject %.1f%% of golden positives (at %.2f)",
		100*record.AtCurrent.FalseApproval.Value, 100*record.AtCurrent.FalseRejection.Value, record.AtCurrent.Threshold))
}
