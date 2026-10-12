package jeweler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type migrationApplierFixture struct {
	destination map[string]string
	calls       int
}

type rollbackApplierFixture struct {
	calls int
}

func (f *rollbackApplierFixture) RestorePrevious(_ context.Context, _ MigrationManifest) error {
	f.calls++
	return nil
}

func (f *migrationApplierFixture) ApplyMigration(_ context.Context, manifest MigrationManifest, actions []MigrationAction) error {
	f.calls++
	for _, action := range actions {
		f.applyAction(manifest, action)
	}
	return nil
}

func (f *migrationApplierFixture) applyAction(manifest MigrationManifest, action MigrationAction) {
	if action.Action != "create" {
		return
	}
	for _, entry := range manifest.Entries {
		if entry.Key == action.Key {
			f.destination[entry.Key] = entry.Digest
		}
	}
}

func migrationFixture(entries []InventoryEntry) MigrationManifest {
	return MigrationManifest{
		SchemaVersion:   MigrationSchemaVersion,
		IdempotencyKey:  "migration-1",
		Source:          validIdentity(),
		Target:          ProviderIdentity{Package: "treasure-chest", Version: "0.2.0", API: "knowledge-api/v1", Digest: "sha256:target", Provenance: "lock"},
		Entries:         entries,
		LegacyPreserved: true,
		RollbackRef:     "sha256:legacy",
	}
}

func TestMigrationManifestCanonicalDigestIgnoresEntryOrder(t *testing.T) {
	entries := []InventoryEntry{
		{Key: "jewel-b", Kind: "jewel", SourcePath: "legacy/b.yaml", Digest: "sha256:b", Status: "accepted", Provenance: "legacy"},
		{Key: "jewel-a", Kind: "jewel", SourcePath: "legacy/a.yaml", Digest: "sha256:a", Status: "verified", Provenance: "legacy"},
	}
	first := migrationFixture(entries)
	second := migrationFixture([]InventoryEntry{entries[1], entries[0]})

	firstJSON, err := first.CanonicalJSON()
	require.NoError(t, err)
	secondJSON, err := second.CanonicalJSON()
	require.NoError(t, err)
	require.JSONEq(t, string(firstJSON), string(secondJSON))

	firstDigest, err := first.Digest()
	require.NoError(t, err)
	secondDigest, err := second.Digest()
	require.NoError(t, err)
	require.Equal(t, firstDigest, secondDigest)
}

func TestMigrationManifestRequiresReplayInventoryFields(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{{Key: "jewel-1", Kind: "jewel", Digest: "sha256:item", Provenance: "legacy"}})
	_, err := manifest.CanonicalJSON()
	require.ErrorContains(t, err, "source_path, digest, status")
}

func TestPlanMigrationSortsActionsByStableKey(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{
		{Key: "jewel-b", Kind: "jewel", SourcePath: "legacy/b.yaml", Digest: "sha256:b", Status: "accepted", Provenance: "legacy"},
		{Key: "jewel-a", Kind: "jewel", SourcePath: "legacy/a.yaml", Digest: "sha256:a", Status: "accepted", Provenance: "legacy"},
	})
	actions, err := PlanMigration(manifest, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"jewel-a", "jewel-b"}, []string{actions[0].Key, actions[1].Key})
}

func TestExecuteMigrationDryRunNeverMutates(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{{Key: "jewel-1", Kind: "jewel", SourcePath: "jewels.yaml", Digest: "sha256:item", Status: "accepted", Provenance: "legacy"}})
	applier := &migrationApplierFixture{destination: map[string]string{}}

	run, err := ExecuteMigration(context.Background(), manifest, applier.destination, true, false, applier)
	require.NoError(t, err)
	require.True(t, run.DryRun)
	require.False(t, run.Applied)
	require.Equal(t, "create", run.Actions[0].Action)
	require.Zero(t, applier.calls)
	require.Empty(t, applier.destination)
}

func TestExecuteMigrationRequiresApprovalAndRejectsConflicts(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{{Key: "jewel-1", Kind: "jewel", SourcePath: "jewels.yaml", Digest: "sha256:item", Status: "accepted", Provenance: "legacy"}})
	applier := &migrationApplierFixture{destination: map[string]string{"jewel-1": "sha256:other"}}

	_, err := ExecuteMigration(context.Background(), manifest, applier.destination, false, false, applier)
	require.ErrorIs(t, err, ErrMigrationApprovalRequired)
	require.Zero(t, applier.calls)
	_, err = ExecuteMigration(context.Background(), manifest, applier.destination, false, true, applier)
	require.ErrorIs(t, err, ErrMigrationConflict)
	require.Zero(t, applier.calls)
}

func TestExecuteMigrationIsIdempotentAfterApprovedApply(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{{Key: "jewel-1", Kind: "jewel", SourcePath: "jewels.yaml", Digest: "sha256:item", Status: "accepted", Provenance: "legacy"}})
	applier := &migrationApplierFixture{destination: map[string]string{}}

	first, err := ExecuteMigration(context.Background(), manifest, applier.destination, false, true, applier)
	require.NoError(t, err)
	require.True(t, first.Applied)
	require.False(t, first.AlreadyApplied)
	require.Equal(t, 1, applier.calls)

	second, err := ExecuteMigration(context.Background(), manifest, applier.destination, false, true, applier)
	require.NoError(t, err)
	require.False(t, second.Applied)
	require.True(t, second.AlreadyApplied)
	require.Equal(t, 1, applier.calls)
}

func TestExecuteMigrationFailsClosedWithoutExternalApplier(t *testing.T) {
	manifest := migrationFixture([]InventoryEntry{{Key: "jewel-1", Kind: "jewel", SourcePath: "jewels.yaml", Digest: "sha256:item", Status: "accepted", Provenance: "legacy"}})

	_, err := ExecuteMigration(context.Background(), manifest, nil, false, true, nil)
	require.ErrorIs(t, err, ErrMigrationApplierUnavailable)
}

func TestExecuteRollbackRestoresWhenParityIsNotVerified(t *testing.T) {
	manifest := migrationFixture(nil)
	applier := &rollbackApplierFixture{}

	run, err := ExecuteRollback(context.Background(), manifest, false, true, applier)
	require.NoError(t, err)
	require.True(t, run.RestoreRequired)
	require.True(t, run.Restored)
	require.True(t, run.LegacyPreserved)
	require.Equal(t, manifest.Source, run.PreviousIdentity)
	require.Equal(t, 1, applier.calls)
}

func TestExecuteRollbackRestoresWhenExternalReadinessIsNotVerified(t *testing.T) {
	manifest := migrationFixture(nil)
	applier := &rollbackApplierFixture{}

	run, err := ExecuteRollback(context.Background(), manifest, true, false, applier)
	require.NoError(t, err)
	require.True(t, run.RestoreRequired)
	require.True(t, run.Restored)
	require.Equal(t, 1, applier.calls)
}

func TestExecuteRollbackIsNoOpForHealthyCutover(t *testing.T) {
	manifest := migrationFixture(nil)

	run, err := ExecuteRollback(context.Background(), manifest, true, true, nil)
	require.NoError(t, err)
	require.False(t, run.RestoreRequired)
	require.False(t, run.Restored)
	require.True(t, run.LegacyPreserved)
}

func TestExecuteRollbackFailsClosedWithoutExternalApplier(t *testing.T) {
	manifest := migrationFixture(nil)

	run, err := ExecuteRollback(context.Background(), manifest, false, true, nil)
	require.ErrorIs(t, err, ErrRollbackApplierUnavailable)
	require.True(t, run.RestoreRequired)
	require.False(t, run.Restored)
	require.True(t, run.LegacyPreserved)
}
