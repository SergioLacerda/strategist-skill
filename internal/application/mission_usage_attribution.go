package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func resolveUsagePhaseRun(root, missionID, phase, runID string, resolve func(string, string, string) (string, error)) (string, error) {
	if runID != "" || resolve == nil {
		return runID, nil
	}
	canonical, ok := domain.ParseUsagePhase(phase)
	if !ok || canonical == domain.UsagePhaseUnattributed {
		return runID, nil
	}
	resolved, err := resolve(root, missionID, canonical)
	if err != nil {
		return "", fmt.Errorf("resolve phase run for %s: %w", canonical, err)
	}
	return resolved, nil
}
