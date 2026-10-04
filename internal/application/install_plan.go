package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// InstallPlanInput is the consumer-owned input to the read-only installation
// planner. Activation, journaling, and rollback are intentionally absent.
type InstallPlanInput struct {
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
func PlanInstall(input InstallPlanInput) (domain.InstallPlan, error) {
	plan, err := domain.NewInstallPlan(
		input.Stage, input.Mode, input.BasePath, input.Slots, input.SlotModes,
		input.Lock, input.Bindings,
	)
	if err != nil {
		return domain.InstallPlan{}, fmt.Errorf("application install planning: %w", err)
	}
	return plan, nil
}
