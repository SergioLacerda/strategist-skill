package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// approvalGatePackageDigest derives, while the mission lock is held, the digest
// of the refined package the user is accepting at the main Approval Gate. Only
// gate_approved binds a digest; every other event, and a mission with no
// refined package yet, yields "" (the handoff evaluation then requires the gate
// to be accepted again). A package that exists but cannot be digested rejects
// the event rather than approving something unidentifiable.
func approvalGatePackageDigest(basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (string, error) {
	if event != domain.MissionEventGateApproved {
		return "", nil
	}
	refined := filepath.Join(basePath, "refined", status.MissionID)
	if _, err := os.Stat(filepath.Join(refined, "analysis.md")); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	digest, err := handoff.PackageDigest(refined)
	if err != nil {
		return "", fmt.Errorf("approval gate package digest: %w", err)
	}
	return digest, nil
}
