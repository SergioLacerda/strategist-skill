// Package installplan owns the read-only application installation planner.
package installplan

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domaininstallplan "github.com/SergioLacerda/strategist-skill/internal/domain/installplan"
)

// Input is the consumer-owned input to the read-only installation
// planner. Activation, journaling, and rollback are intentionally absent.
type Input struct {
	Stage     domain.Stage
	Mode      string
	BasePath  string
	Slots     map[string]string
	SlotModes map[string]string
	Lock      domain.PluginLock
	Bindings  []domain.SlotBinding
}

// PlanInstall creates the single deterministic InstallPlan used by interactive
// Wizard and silent installation adapters.
func PlanInstall(input Input) (domaininstallplan.InstallPlan, error) {
	plan, err := domaininstallplan.NewInstallPlan(
		input.Stage, input.Mode, input.BasePath, input.Slots, input.SlotModes,
		input.Lock, input.Bindings,
	)
	if err != nil {
		return domaininstallplan.InstallPlan{}, fmt.Errorf("application install planning: %w", err)
	}
	return plan, nil
}
