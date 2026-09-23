package install

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

// UpgradeService plans and applies a runtime upgrade; production wires
// internal/install.Service, tests supply a fake.
type UpgradeService interface {
	PlanUpgrade(strategistDir string) (internalinstall.UpgradePlan, error)
	ApplyUpgrade(strategistDir string, plan internalinstall.UpgradePlan, force bool) (backupDir string, err error)
}

// UpgradeDependencies supplies host-specific concerns for the upgrade command.
type UpgradeDependencies struct {
	ResolveTarget  func(explicit string, global bool) (string, error)
	ServiceFactory func(allowDowngrade bool) UpgradeService
}

type upgradeOptions struct {
	Target, Rollback                      string
	Global, DryRun, Force, AllowDowngrade bool
}

// NewUpgrade creates an isolated upgrade command with no package-level flag state.
func NewUpgrade(deps UpgradeDependencies) *cobra.Command {
	opts := upgradeOptions{}
	cmd := &cobra.Command{
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
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.Target, "target", "", "target repository root (default: current directory)")
	flags.BoolVar(&opts.Global, "global", false, "operate on the global root (default: local project)")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "show the upgrade plan without writing anything")
	flags.BoolVar(&opts.Force, "force", false, "also overwrite customized files (default: preserve them)")
	flags.BoolVar(&opts.AllowDowngrade, "allow-downgrade", false, "let this binary replace normative files installed by a newer binary (deliberate rollback; default: refuse with runtime_newer_than_binary)")
	flags.StringVar(&opts.Rollback, "rollback", "", `restore files from a previous upgrade's backup instead of upgrading ("latest" or a specific timestamp from .strategist/.upgrade-backups/)`)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return runUpgrade(cmd, deps, opts) }
	return cmd
}

func runUpgrade(cmd *cobra.Command, deps UpgradeDependencies, opts upgradeOptions) error {
	if deps.ResolveTarget == nil {
		return fmt.Errorf("upgrade: target resolver is not configured")
	}
	if deps.ServiceFactory == nil {
		return fmt.Errorf("upgrade: service factory is not configured")
	}
	target, err := deps.ResolveTarget(opts.Target, opts.Global)
	if err != nil {
		return err
	}
	strategistDir := filepath.Join(target, ".strategist")

	if opts.Rollback != "" {
		return runUpgradeRollback(cmd, strategistDir, opts.Rollback)
	}
	return executeUpgrade(cmd, deps.ServiceFactory(opts.AllowDowngrade), strategistDir, opts)
}

func executeUpgrade(cmd *cobra.Command, svc UpgradeService, strategistDir string, opts upgradeOptions) error {
	plan, err := svc.PlanUpgrade(strategistDir)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	out := cmd.OutOrStdout()
	if err := printUpgradePlan(out, plan, opts.Force); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	if opts.DryRun {
		return writeUpgradeDryRun(out)
	}

	backupDir, err := svc.ApplyUpgrade(strategistDir, plan, opts.Force)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	if err := writeUpgradeBackupNotice(out, backupDir); err != nil {
		return err
	}
	return writeUpgradeComplete(out)
}

func writeUpgradeDryRun(out io.Writer) error {
	if _, err := fmt.Fprintln(out, "\n(dry run — nothing written)"); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func writeUpgradeBackupNotice(out io.Writer, backupDir string) error {
	if backupDir == "" {
		return nil
	}
	if _, err := fmt.Fprintf(out, "\nBacked up overwritten files to %s\n", backupDir); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	return nil
}

func writeUpgradeComplete(out io.Writer) error {
	if _, err := fmt.Fprintln(out, "Upgrade complete."); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func runUpgradeRollback(cmd *cobra.Command, strategistDir, stamp string) error {
	if stamp == "latest" {
		stamps, err := internalinstall.ListUpgradeBackups(strategistDir)
		if err != nil {
			return fmt.Errorf("upgrade rollback: %w", err)
		}
		if len(stamps) == 0 {
			return fmt.Errorf("upgrade rollback: no backups found under %s/.upgrade-backups", strategistDir)
		}
		stamp = stamps[0]
	}
	count, err := internalinstall.RollbackUpgrade(strategistDir, stamp)
	if err != nil {
		return fmt.Errorf("upgrade rollback: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Restored %d file(s) from backup %s.\n", count, stamp); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func printUpgradePlan(out io.Writer, plan internalinstall.UpgradePlan, force bool) error {
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
