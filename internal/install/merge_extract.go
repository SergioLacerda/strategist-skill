package install

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Report carries facts about an Install run that aren't errors but
// are worth reporting to the user — currently just BackupDir, populated
// when the three-way merge path (extractRuntimeTree) overwrote at least one
// AutoUpgrade/Customized file and snapshotted it first. Empty when nothing
// was overwritten, or on the s.Lister==nil legacy fallback path.
type Report struct {
	BackupDir string
}

// prepareRuntime extracts the embedded runtime tree into strategistDir and
// applies the strictly-guarded normative-file plan on top. It returns the
// paths it created (for rollback bookkeeping), the full embedded path->hash
// map for the manifest finalizeInstall writes when the three-way merge path
// ran (see extractRuntimeTree), and that path's backup dir (empty if no
// file was overwritten, or on the legacy fallback path).
func (s Service) prepareRuntime(ctx context.Context, strategistDir string, cfg domain.InstallConfig, plan runtimeDefaultPlan) ([]string, map[string]string, string, error) {
	slog.InfoContext(ctx, "[Strategist] install extracting-defaults",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(strategistDir),
	)
	fullHashes, backupDir, err := s.extractRuntimeTree(strategistDir, cfg.Force)
	if err != nil {
		return nil, nil, "", fmt.Errorf("install: extract defaults: %w", err)
	}
	if err := s.applyRuntimeDefaultPlan(ctx, strategistDir, plan); err != nil {
		return nil, nil, "", err
	}
	return []string{strategistDir}, fullHashes, backupDir, nil
}

// extractRuntimeTree populates strategistDir from the embedded defaults and
// returns the full embedded path->hash map for the current release (nil when
// that map is unavailable — see below), so the caller can persist a
// full-tree install manifest.
//
// When s.Lister is configured (every real `strategist install` invocation
// wires embed.Extractor as both Extractor and Lister), this uses the same
// three-way file-state classification `strategist upgrade` uses
// (domain.DecideUpgradeFileState / domain.UpgradeFileWillWrite) across the
// *entire* embedded tree, consulting the prior install manifest written by a
// previous install or upgrade: a file that still matches the exact default it
// was installed with is transparently updated to the new embedded version
// (UpgradeAutoUpgrade), a file the user has actually modified is preserved
// unless force is set (UpgradeCustomized), and a genuinely new file is
// written (UpgradeMissing). This closes the gap the plain raw-merge
// extractor (embed.Extractor.Extract) could not: it only ever compares
// on-disk content against the *current* embedded default, so it cannot tell
// "user kept the old default unmodified" from "user customized it" — both
// simply differ from the new embedded content and were silently preserved.
//
// When s.Lister is nil (older/minimal Service configurations — some test
// doubles only implement domain.FileExtractor), this falls back to the
// legacy s.Extractor.Extract raw-merge behavior unchanged, so every existing
// caller that never wired a Lister keeps working exactly as before.
// extractRuntimeTree's third return value is the backup dir created by the
// three-way path when it overwrote at least one AutoUpgrade/Customized file
// (empty when nothing was overwritten, or when the s.Lister==nil legacy
// fallback ran) — callers that want to report it to the user (see
// Service.InstallWithReport) get it directly instead of needing to guess
// whether a snapshot happened.
func (s Service) extractRuntimeTree(strategistDir string, force bool) (fullHashes map[string]string, backupDir string, err error) {
	if s.Lister == nil {
		if err := s.Extractor.Extract(strategistDir, force); err != nil {
			return nil, "", fmt.Errorf("extract runtime tree: %w", err)
		}
		return nil, "", nil
	}

	plan, err := s.PlanUpgrade(strategistDir)
	if err != nil {
		return nil, "", fmt.Errorf("extract runtime tree: %w", err)
	}

	// Reuse ApplyUpgrade's own write set + backup-before-overwrite logic
	// (upgradeWriteSet/snapshotBeforeUpgrade/writeUpgradeFile) instead of
	// reimplementing the write loop — a prior version of this function wrote
	// AutoUpgrade/Customized files directly via writeUpgradeFile with no
	// snapshot step, so `strategist install --force` in merge mode could
	// silently overwrite a user's customized file with no backup, unlike
	// `strategist upgrade --force`'s identical operation. This intentionally
	// does not call ApplyUpgrade itself, since ApplyUpgrade also writes its
	// own full install manifest — this function's caller (prepareRuntime)
	// already returns plan.embeddedHashes for finalizeInstall to persist the
	// manifest once, and calling ApplyUpgrade here would write it twice.
	toWrite, toBackup := upgradeWriteSet(plan, force)
	if len(toBackup) > 0 {
		backupDir, err = s.snapshotBeforeUpgrade(strategistDir, toBackup)
		if err != nil {
			return nil, "", fmt.Errorf("extract runtime tree: %w", err)
		}
	}
	for _, p := range toWrite {
		if err := s.writeUpgradeFile(strategistDir, p); err != nil {
			return nil, "", fmt.Errorf("extract runtime tree: %w", err)
		}
	}

	return plan.embeddedHashes, backupDir, nil
}

// buildInstallManifest chooses the install manifest shape to persist.
// fullHashes non-nil means extractRuntimeTree ran the three-way path (a
// Lister was configured) and covered the entire embedded tree, so the
// manifest should too — a future install/upgrade needs every path recorded,
// not just the small normative subset, to correctly classify auto-upgrade
// vs. customized. fullHashes nil means the legacy raw-extractor fallback ran
// (no Lister configured), so the manifest keeps its historical narrower
// (normative-only) shape for backward compatibility.
func buildInstallManifest(pkgID string, plan runtimeDefaultPlan, fullHashes map[string]string) domain.InstallManifest {
	if fullHashes != nil {
		return domain.NewFullInstallManifest(pkgID, fullHashes)
	}
	return domain.NewInstallManifest(pkgID, plan.embeddedHashes)
}
