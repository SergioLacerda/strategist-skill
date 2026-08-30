package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

var (
	upgradeTarget   string
	upgradeGlobal   bool
	upgradeDryRun   bool
	upgradeForce    bool
	upgradeRollback string
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Reconcile an installed .strategist/ runtime against the current embedded defaults",
	Long: `Reconcile an installed .strategist/ runtime against the current embedded
defaults across the full runtime tree (unlike a plain "strategist install",
which only re-applies the small set of strictly-guarded normative files).

Every file is classified before anything is written:
  managed      matches the current embedded default already — no-op
  missing      part of current defaults, absent on disk — written
  auto_upgrade on disk, matches what was last installed, embedded moved on — written
  customized   on disk, diverges from what was last installed — preserved unless --force
  orphaned     tracked by a prior install, no longer part of current defaults — reported, never deleted

Use --dry-run to see the plan without writing anything. Any file upgrade
overwrites (auto_upgrade, or customized with --force) is snapshotted first
under .strategist/.upgrade-backups/<timestamp>/ — restore it with
"strategist upgrade --rollback <timestamp>" (or --rollback latest).`,
	RunE: runUpgrade,
}

func runUpgrade(cmd *cobra.Command, _ []string) error {
	target, err := resolveInstallTarget(upgradeTarget, upgradeGlobal)
	if err != nil {
		return err
	}
	strategistDir := filepath.Join(target, ".strategist")

	svc := upgradeService()

	if upgradeRollback != "" {
		return runUpgradeRollback(cmd, strategistDir, upgradeRollback)
	}

	plan, err := svc.PlanUpgrade(strategistDir)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	out := cmd.OutOrStdout()
	if err := printUpgradePlan(out, plan, upgradeForce); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	if upgradeDryRun {
		if _, err := fmt.Fprintln(out, "\n(dry run — nothing written)"); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}

	backupDir, err := svc.ApplyUpgrade(strategistDir, plan, upgradeForce)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	if backupDir != "" {
		if _, err := fmt.Fprintf(out, "\nBacked up overwritten files to %s\n", backupDir); err != nil {
			return fmt.Errorf("upgrade: %w", err)
		}
	}
	if _, err := fmt.Fprintln(out, "Upgrade complete."); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func runUpgradeRollback(cmd *cobra.Command, strategistDir, stamp string) error {
	if stamp == "latest" {
		stamps, err := install.ListUpgradeBackups(strategistDir)
		if err != nil {
			return fmt.Errorf("upgrade rollback: %w", err)
		}
		if len(stamps) == 0 {
			return fmt.Errorf("upgrade rollback: no backups found under %s/.upgrade-backups", strategistDir)
		}
		stamp = stamps[0]
	}
	count, err := install.RollbackUpgrade(strategistDir, stamp)
	if err != nil {
		return fmt.Errorf("upgrade rollback: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Restored %d file(s) from backup %s.\n", count, stamp); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func printUpgradePlan(out io.Writer, plan install.UpgradePlan, force bool) error {
	byState := map[domain.RuntimeFileUpgradeState][]string{}
	for _, e := range plan.Entries {
		byState[e.State] = append(byState[e.State], e.Path)
	}

	if _, err := fmt.Fprintf(out, "managed (no change):     %d\n", len(byState[domain.UpgradeManaged])); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := printUpgradeGroup(out, "missing (will write)", byState[domain.UpgradeMissing]); err != nil {
		return err
	}
	if err := printUpgradeGroup(out, "auto_upgrade (will write)", byState[domain.UpgradeAutoUpgrade]); err != nil {
		return err
	}
	label := "customized (preserved)"
	if force {
		label = "customized (will OVERWRITE — --force)"
	}
	if err := printUpgradeGroup(out, label, byState[domain.UpgradeCustomized]); err != nil {
		return err
	}
	return printUpgradeGroup(out, "orphaned (not deleted — review manually)", byState[domain.UpgradeOrphaned])
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

func upgradeService() install.Service {
	return install.Service{
		Extractor: embedpkg.Extractor{},
		Lister:    embedpkg.Extractor{},
		Version:   Version,
	}
}

func init() {
	upgradeCmd.Flags().StringVar(&upgradeTarget, "target", "", "target repository root (default: current directory)")
	upgradeCmd.Flags().BoolVar(&upgradeGlobal, "global", false, "operate on the global root (default: local project)")
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "show the upgrade plan without writing anything")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "also overwrite customized files (default: preserve them)")
	upgradeCmd.Flags().StringVar(&upgradeRollback, "rollback", "", `restore files from a previous upgrade's backup instead of upgrading ("latest" or a specific timestamp from .strategist/.upgrade-backups/)`)
}
