package jeweler

import (
	"context"
	"fmt"
	"sort"
)

// MigrationAction is a deterministic dry-run result. Applying it belongs to
// the explicitly authorized maintenance provider, never to consumer queries.
type MigrationAction struct {
	Key    string `json:"key"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// MigrationApplier is the external provider's authorized mutation seam. The
// host can plan and validate a migration, but it never writes provider data
// directly. Implementations must honor manifest.IdempotencyKey and preserve
// the legacy source named by the manifest.
type MigrationApplier interface {
	ApplyMigration(context.Context, MigrationManifest, []MigrationAction) error
}

// RollbackApplier is the external provider's recovery seam. It restores the
// previous provider or binding identified by manifest.Source; it must not
// delete or rewrite legacy data. Implementations must make replay by the
// manifest digest idempotent.
type RollbackApplier interface {
	RestorePrevious(context.Context, MigrationManifest) error
}

// MigrationRun is the auditable result of one dry-run or approved execution.
// Applied is false for dry-runs and for idempotent replays that found every
// destination digest already present.
type MigrationRun struct {
	ManifestDigest  string            `json:"manifest_digest"`
	Actions         []MigrationAction `json:"actions"`
	DryRun          bool              `json:"dry_run"`
	Approved        bool              `json:"approved"`
	Applied         bool              `json:"applied"`
	AlreadyApplied  bool              `json:"already_applied"`
	LegacyPreserved bool              `json:"legacy_preserved"`
}

// ExecuteMigration performs validation and a deterministic dry-run, then
// delegates an approved, conflict-free mutation to the external provider.
// Passing dryRun=true never calls applier. Passing dryRun=false without
// approved=true or without an applier fails closed before any mutation.
func ExecuteMigration(ctx context.Context, manifest MigrationManifest, destination map[string]string, dryRun, approved bool, applier MigrationApplier) (MigrationRun, error) {
	digest, err := manifest.Digest()
	if err != nil {
		return MigrationRun{}, err
	}
	actions, err := PlanMigration(manifest, destination)
	if err != nil {
		return MigrationRun{}, err
	}
	run := MigrationRun{ManifestDigest: digest, Actions: actions, DryRun: dryRun, Approved: approved, LegacyPreserved: manifest.LegacyPreserved}
	if dryRun {
		return run, nil
	}
	return applyMigration(ctx, manifest, actions, approved, applier, run)
}

func applyMigration(ctx context.Context, manifest MigrationManifest, actions []MigrationAction, approved bool, applier MigrationApplier, run MigrationRun) (MigrationRun, error) {
	if !approved {
		return run, ErrMigrationApprovalRequired
	}
	if hasMigrationConflicts(actions) {
		return run, ErrMigrationConflict
	}
	if allMigrationActionsPreserve(actions) {
		run.AlreadyApplied = true
		return run, nil
	}
	if applier == nil {
		return run, ErrMigrationApplierUnavailable
	}
	if err := applier.ApplyMigration(ctx, manifest, actions); err != nil {
		return run, fmt.Errorf("jeweler: apply migration %s: %w", manifest.IdempotencyKey, err)
	}
	run.Applied = true
	return run, nil
}

// RollbackRun is the auditable result of a recovery attempt. LegacyPreserved
// remains true because rollback never authorizes destructive cleanup of the
// legacy source.
type RollbackRun struct {
	ManifestDigest   string           `json:"manifest_digest"`
	PreviousIdentity ProviderIdentity `json:"previous_identity"`
	RestoreRequired  bool             `json:"restore_required"`
	Restored         bool             `json:"restored"`
	LegacyPreserved  bool             `json:"legacy_preserved"`
}

// ExecuteRollback restores the previous binding only when parity or external
// readiness is not verified. A healthy cutover is a no-op; an unhealthy one
// fails closed when no authorized external recovery adapter is available.
func ExecuteRollback(ctx context.Context, manifest MigrationManifest, parityVerified, externalReady bool, applier RollbackApplier) (RollbackRun, error) {
	digest, err := manifest.Digest()
	if err != nil {
		return RollbackRun{}, err
	}
	run := RollbackRun{ManifestDigest: digest, PreviousIdentity: manifest.Source, RestoreRequired: !parityVerified || !externalReady, LegacyPreserved: manifest.LegacyPreserved}
	if !run.RestoreRequired {
		return run, nil
	}
	if applier == nil {
		return run, ErrRollbackApplierUnavailable
	}
	if err := applier.RestorePrevious(ctx, manifest); err != nil {
		return run, fmt.Errorf("jeweler: rollback %s: %w", manifest.IdempotencyKey, err)
	}
	run.Restored = true
	return run, nil
}

// PlanMigration compares a validated manifest with destination digests.
func PlanMigration(manifest MigrationManifest, destination map[string]string) ([]MigrationAction, error) {
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	canonical := append([]InventoryEntry(nil), manifest.Entries...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Key < canonical[j].Key })
	actions := make([]MigrationAction, 0, len(canonical))
	for _, entry := range canonical {
		digest, exists := destination[entry.Key]
		switch {
		case !exists:
			actions = append(actions, MigrationAction{Key: entry.Key, Action: "create", Reason: "destination_missing"})
		case digest == entry.Digest:
			actions = append(actions, MigrationAction{Key: entry.Key, Action: "preserve", Reason: "digest_match"})
		default:
			actions = append(actions, MigrationAction{Key: entry.Key, Action: "conflict", Reason: "destination_digest_differs"})
		}
	}
	return actions, nil
}

func hasMigrationConflicts(actions []MigrationAction) bool {
	for _, action := range actions {
		if action.Action == "conflict" {
			return true
		}
	}
	return false
}

func allMigrationActionsPreserve(actions []MigrationAction) bool {
	if len(actions) == 0 {
		return true
	}
	for _, action := range actions {
		if action.Action != "preserve" {
			return false
		}
	}
	return true
}
