package governance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type governanceCore struct {
	Items []struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	} `json:"items"`
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

func readSDDMetadata(metaPath string) (SDDMetadata, error) {
	metaRaw, err := os.ReadFile(metaPath) //nolint:gosec // G304: metadata path is derived from the configured .sdd directory
	if os.IsNotExist(err) {
		return SDDMetadata{}, fmt.Errorf(".sdd/metadata.json not found — is SDD active in this workspace? (path: %s)", metaPath)
	}
	if err != nil {
		return SDDMetadata{}, fmt.Errorf("read metadata: %w", err)
	}
	var meta SDDMetadata
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return SDDMetadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

func readGovernanceCore(corePath string) (governanceCore, error) {
	coreRaw, err := os.ReadFile(corePath) //nolint:gosec // G304: governance core path is derived from the configured .sdd directory
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
