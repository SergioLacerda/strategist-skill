package check

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func checkProviders(cfg domain.ActiveConfig) map[string]string {
	return map[string]string{
		"discovery":  cfg.Slots["discovery"],
		"refinement": cfg.Slots["refinement"],
		"execution":  cfg.Slots["execution"],
	}
}

func checkDecisionReason(providers map[string]string, errs []string) string {
	decisionReason := "all_slots_ready"
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		if providers[slot] == "" {
			return "slot_provider_missing:" + slot
		}
	}
	if len(errs) > 0 {
		decisionReason = "validation_failed"
	}
	return decisionReason
}
