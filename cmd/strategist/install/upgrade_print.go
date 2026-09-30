package install

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
)

func printUpgradePlan(out io.Writer, plan internalinstall.UpgradePlan, force bool) error {
	byState := map[domain.RuntimeFileUpgradeState][]string{}
	for _, e := range plan.Entries {
		byState[e.State] = append(byState[e.State], e.Path)
	}

	if _, err := fmt.Fprintf(out, "managed (no change):     %d\n", len(byState[domain.UpgradeManaged])); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	customized := "customized (preserved)"
	if force {
		customized = "customized (will OVERWRITE — --force)"
	}
	groups := []struct {
		label string
		state domain.RuntimeFileUpgradeState
	}{
		{"missing (will write)", domain.UpgradeMissing},
		{"auto_upgrade (will write)", domain.UpgradeAutoUpgrade},
		{customized, domain.UpgradeCustomized},
		{"legacy_layout (will migrate: snapshot, then remove)", domain.UpgradeLegacyLayout},
		{"orphaned (not deleted — review manually)", domain.UpgradeOrphaned},
	}
	for _, group := range groups {
		if err := printUpgradeGroup(out, group.label, byState[group.state]); err != nil {
			return err
		}
	}
	return printLockMigration(out, plan.LockMigration)
}

// printLockMigration names the slots whose plugins.lock Ranked binding gains a
// weapon_version when the upgrade is applied.
func printLockMigration(out io.Writer, slots []string) error {
	if len(slots) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(out, "plugins.lock (will migrate: weapon_version): %s\n", strings.Join(slots, ", ")); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func printUpgradeGroup(out io.Writer, label string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	sort.Strings(paths)
	if _, err := fmt.Fprintf(out, "%s: %d\n", label, len(paths)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, p := range paths {
		if _, err := fmt.Fprintf(out, "  - %s\n", p); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}
