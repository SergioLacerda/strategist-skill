package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type syncReport struct {
	GovernanceFingerprint string
	MandatesActive        []string
	MandatesCompliant     []string
	MandatesPartial       []string
	MandatesMissing       []string
	FieldsApplied         []string
	DryRun                bool
}

type sddMetadata struct {
	Version      string `json:"version"`
	Fingerprints struct {
		Combined string `json:"combined"`
	} `json:"fingerprints"`
	GovernanceFingerprint string            `json:"governance_fingerprint"` // fallback field name
	Mandates              map[string]string `json:"mandates"`
}

type governanceCore struct {
	Items []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	} `json:"items"`
}

func runSyncGovernance(skillRoot, sddDir string, dryRun bool) (syncReport, error) {
	report := syncReport{DryRun: dryRun}

	fp, activeMandates, err := readGovernance(sddDir)
	if err != nil {
		return report, err
	}
	report.GovernanceFingerprint = fp
	report.MandatesActive = activeMandates

	skillPath := filepath.Join(skillRoot, "skill.yaml")
	skill, err := readSkill(skillPath)
	if err != nil {
		return report, err
	}

	computeComplianceGaps(&report, skill)
	changed := applyMissingFields(skill, &report)

	if changed && !dryRun {
		if err := writeSyncedSkill(skillPath, skill); err != nil {
			return report, err
		}
	}

	return report, nil
}

func writeSyncedSkill(skillPath string, skill map[string]any) error {
	out, err := yaml.Marshal(skill)
	if err != nil {
		return fmt.Errorf("marshal skill.yaml: %w", err)
	}
	if err := os.WriteFile(skillPath, out, 0o644); err != nil {
		return fmt.Errorf("write skill.yaml: %w", err)
	}
	return nil
}

func readGovernance(sddDir string) (fingerprint string, activeMandates []string, err error) {
	metaPath := filepath.Join(sddDir, "metadata.json")
	meta, err := readSDDMetadata(metaPath)
	if err != nil {
		return "", nil, err
	}
	fp := meta.Fingerprints.Combined
	if fp == "" {
		fp = meta.GovernanceFingerprint
	}

	corePath := filepath.Join(sddDir, "source", "governance-core.json")
	core, err := readGovernanceCore(corePath)
	if err != nil {
		return "", nil, err
	}
	return fp, activeMandateIDs(core), nil
}

func readSDDMetadata(metaPath string) (sddMetadata, error) {
	metaRaw, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return sddMetadata{}, fmt.Errorf(".sdd/metadata.json not found — is SDD active in this workspace? (path: %s)", metaPath)
	}
	if err != nil {
		return sddMetadata{}, fmt.Errorf("read metadata: %w", err)
	}
	var meta sddMetadata
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return sddMetadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

func readGovernanceCore(corePath string) (governanceCore, error) {
	coreRaw, err := os.ReadFile(corePath)
	if err != nil {
		return governanceCore{}, fmt.Errorf("read governance-core.json: %w", err)
	}
	var core governanceCore
	if err := json.Unmarshal(coreRaw, &core); err != nil {
		return governanceCore{}, fmt.Errorf("parse governance-core.json: %w", err)
	}
	return core, nil
}

func activeMandateIDs(core governanceCore) []string {
	var activeMandates []string
	for _, item := range core.Items {
		if item.Type == "MANDATE" && item.Status == "required" {
			activeMandates = append(activeMandates, item.ID)
		}
	}
	return activeMandates
}

func readSkill(skillPath string) (map[string]any, error) {
	skillRaw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("read skill.yaml: %w", err)
	}
	var skill map[string]any
	if err := yaml.Unmarshal(skillRaw, &skill); err != nil {
		return nil, fmt.Errorf("parse skill.yaml: %w", err)
	}
	return skill, nil
}
