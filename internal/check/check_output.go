package check

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func printCheckSuccess(root string, providers map[string]string, resolutions map[string]slotResolution, mode string, policy domain.ResolutionPolicy, weaponBindings []weaponBinding) error {
	printStatusBanner("check")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, section := range []func(*tabwriter.Writer) error{
		func(w *tabwriter.Writer) error { return writeCheckStatusSection(w, root) },
		func(w *tabwriter.Writer) error { return writeCheckSlotsSection(w, providers, resolutions, policy) },
		func(w *tabwriter.Writer) error { return writeCheckReadinessSection(w, resolutions) },
		func(w *tabwriter.Writer) error { return writeCheckWeaponLinksSection(w, weaponBindings) },
		func(w *tabwriter.Writer) error { return writeCheckPersonaSection(w, mode) },
		func(w *tabwriter.Writer) error { return writeCheckPolicySection(w, policy) },
	} {
		if err := section(w); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("check: flush output: %w", err)
	}
	return nil
}

func writeCheckReadinessSection(w *tabwriter.Writer, resolutions map[string]slotResolution) error {
	if _, err := fmt.Fprintln(w, "READINESS\t"); err != nil {
		return fmt.Errorf("check: write readiness header: %w", err)
	}
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		vector := resolutions[slot].readiness
		status := "not_ready"
		if vector.Ready() {
			status = "ready"
		}
		reasons := vector.ReasonCodes()
		reasonText := "none"
		if len(reasons) > 0 {
			reasonText = fmt.Sprintf("%v", reasons)
		}
		if _, err := fmt.Fprintf(w, "  %-12s\t%s\treasons=%s\n", slot, status, reasonText); err != nil {
			return fmt.Errorf("check: write readiness row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeCheckStatusSection(w *tabwriter.Writer, root string) error {
	if _, err := fmt.Fprintln(w, "STATUS\t"); err != nil {
		return fmt.Errorf("check: write status header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  ok\troot=%s\n", root); err != nil {
		return fmt.Errorf("check: write status row: %w", err)
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeCheckSlotsSection(w *tabwriter.Writer, providers map[string]string, resolutions map[string]slotResolution, policy domain.ResolutionPolicy) error {
	if _, err := fmt.Fprintln(w, "SLOTS\t"); err != nil {
		return fmt.Errorf("check: write slots header: %w", err)
	}
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		res := resolutions[slot]
		row := fmt.Sprintf("  %-12s\t%s\tkind=%s", slot, providers[slot], res.kind.label())
		if res.hasFallback() {
			outcome := domain.DecideSlotFallbackOutcome(slot, policy, true)
			row += fmt.Sprintf("\tfallback=%s(native_role)\toutcome=%s", res.fallbackProvider, outcome)
		}
		if _, err := fmt.Fprintln(w, row); err != nil {
			return fmt.Errorf("check: write slot row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

// writeCheckWeaponLinksSection reports DEC-003's always-run embedded-weapon↔role
// verification (docs/adr/0035-embedded-weapon-fallback-policy.md), independent
// of what active.yaml currently configures for any slot — see
// verifyEmbeddedWeaponBindings. The section is omitted entirely when no
// installed skill declares a canonical_role (nothing to report), rather than
// printing an empty header.
func writeCheckWeaponLinksSection(w *tabwriter.Writer, bindings []weaponBinding) error {
	if len(bindings) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "WEAPON LINKS\t"); err != nil {
		return fmt.Errorf("check: write weapon links header: %w", err)
	}
	for _, b := range bindings {
		status := "ok"
		if !b.OK {
			status = "FAIL: " + b.Reason
		}
		if _, err := fmt.Fprintf(w, "  %s→%s\t%s\n", b.SkillID, b.CanonicalRole, status); err != nil {
			return fmt.Errorf("check: write weapon link row: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeCheckPersonaSection(w *tabwriter.Writer, mode string) error {
	if _, err := fmt.Fprintln(w, "PERSONA\t"); err != nil {
		return fmt.Errorf("check: write persona header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  mode\t%s\n", mode); err != nil {
		return fmt.Errorf("check: write persona row: %w", err)
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

// writeCheckPolicySection reports the effective provider_resolution_policy
// (docs/adr/0028-native-role-resilient-baseline.md) — the policy an agent must
// follow when a skill_provider-resolved slot (see the SLOTS section's
// fallback= annotation) turns out not to be invocable at mission time.
func writeCheckPolicySection(w *tabwriter.Writer, policy domain.ResolutionPolicy) error {
	if _, err := fmt.Fprintln(w, "POLICY\t"); err != nil {
		return fmt.Errorf("check: write policy header: %w", err)
	}
	effective := policy.EffectivePolicy()
	suffix := ""
	if policy == "" {
		suffix = " (default)"
	}
	if _, err := fmt.Fprintf(w, "  provider_resolution\t%s%s\n", effective, suffix); err != nil {
		return fmt.Errorf("check: write policy row: %w", err)
	}
	return nil
}
