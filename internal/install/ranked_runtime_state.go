package install

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func writeRankedRuntimeState(strategistDir string, state domain.RankedRuntimeState) error {
	if len(state.Entries) == 0 {
		return nil
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ranked runtime state: %w", err)
	}
	data = append(data, '\n')
	return atomicWriteFile(filepath.Join(strategistDir, domain.RankedRuntimeStatePath), data, 0o644)
}

func openSpecCommandArgs(command, verb string) ([]string, error) {
	fields := strings.Fields(command)
	if len(fields) < 2 || fields[0] != "openspec" || fields[1] != verb {
		return nil, fmt.Errorf("expected openspec %s command, got %q", verb, command)
	}
	return fields[1:], nil
}
