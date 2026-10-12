package check

import (
	"fmt"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"os"
	"text/tabwriter"
)

func printCheckSuccess(root string, providers map[string]string, resolutions map[string]slotResolution, mode string, weaponBindings []weaponBinding) error {
	printStatusBanner("check")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, section := range []func(*tabwriter.Writer) error{
		func(w *tabwriter.Writer) error { return writeCheckStatusSection(w, root) },
		func(w *tabwriter.Writer) error { return writeCheckSlotsSection(w, providers, resolutions) },
		func(w *tabwriter.Writer) error { return writeCheckReadinessSection(w, resolutions) },
		func(w *tabwriter.Writer) error { return writeCheckWeaponLinksSection(w, weaponBindings) },
		func(w *tabwriter.Writer) error { return writeCheckPersonaSection(w, mode) },
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
	if _, err := fmt.Fprintln(w, "ROLE READINESS\t"); err != nil {
		return fmt.Errorf("check: write readiness header: %w", err)
	}
	registry := domain.DefaultRoleRegistry()
	for _, required := range domain.RequiredSlots() {
		slot := string(required)
		role, _ := registry.RoleForSlot(slot)
		if err := writeReadinessRow(w, slot, role.ID, resolutions); err != nil {
			return err
		}
	}
	return writeReadinessAdvisory(w)
}

func writeReadinessAdvisory(w *tabwriter.Writer) error {
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	if _, err := fmt.Fprintln(w, "ADVISORY\t"); err != nil {
		return fmt.Errorf("check: write advisory header: %w", err)
	}
	if _, err := fmt.Fprintln(w, "  advanced_governance\tdeferred\ttrust/dependencies/grants/connectors/observation"); err != nil {
		return fmt.Errorf("check: write advisory row: %w", err)
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write advisory separator: %w", err)
	}
	return nil
}

func writeReadinessRow(w *tabwriter.Writer, slot, role string, resolutions map[string]slotResolution) error {
	status, reason := "ready", "minimum_contract_valid"
	if _, ok := resolutions[slot]; !ok {
		status, reason = "fatal", "provider_not_resolved"
	}
	if _, err := fmt.Fprintf(w, "  %-12s\t%s\trole=%s\treason=%s\n", slot, status, role, reason); err != nil {
		return fmt.Errorf("check: write readiness row: %w", err)
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

func writeCheckSlotsSection(w *tabwriter.Writer, providers map[string]string, resolutions map[string]slotResolution) error {
	if _, err := fmt.Fprintln(w, "SLOTS\t"); err != nil {
		return fmt.Errorf("check: write slots header: %w", err)
	}
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		if err := writeSlotRow(w, slot, providers[slot], resolutions[slot]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeSlotRow(w *tabwriter.Writer, slot, provider string, res slotResolution) error {
	row := fmt.Sprintf("  %-12s\t%s\tkind=%s", slot, provider, res.kind.label())
	if slot == "discovery" || slot == "refinement" {
		row += "\tbinding=valid"
	}
	if _, err := fmt.Fprintln(w, row); err != nil {
		return fmt.Errorf("check: write slot row: %w", err)
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
		if err := writeWeaponLinkRow(w, b); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\t"); err != nil {
		return fmt.Errorf("check: write separator: %w", err)
	}
	return nil
}

func writeWeaponLinkRow(w *tabwriter.Writer, b weaponBinding) error {
	status := "ok"
	if !b.OK {
		status = "FAIL: " + b.Reason
	}
	if _, err := fmt.Fprintf(w, "  %s→%s\t%s\n", b.SkillID, b.CanonicalRole, status); err != nil {
		return fmt.Errorf("check: write weapon link row: %w", err)
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
