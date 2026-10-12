package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

const runbookSidecarFixture = `schema_version: "1"
runbook_id: m1-runbook
runbook_type: analytical
source_doc: docs/runbooks/m1-runbook.md
applies_when:
  - timeout
objective: fixture
`

func runbookSnapshot(sidecarMeta map[string]string) domain.ExecutionSnapshot {
	return domain.ExecutionSnapshot{
		Version: domain.ExecutionSnapshotVersion, MissionID: "m1",
		Groups: []domain.ExecutionGroup{{
			ID: "runbook-OA-RUNBOOK-m1", Resolver: domain.ResolverKey{Kind: "runbook", Origin: domain.ExecutionTargetOriginAcceptedSideQuest},
			SideQuestID: "OA-RUNBOOK-m1", Kind: "runbook", Origin: domain.ExecutionTargetOriginAcceptedSideQuest,
			CompletionPolicy: domain.ExecutionCompletionAllRequired,
			Artifacts: []domain.ExecutionArtifact{
				{ID: "main", Role: domain.ExecutionArtifactRolePrimary, Path: "docs/runbooks/m1-runbook.md", Kind: "runbook"},
				{ID: "sidecar", Role: domain.ExecutionArtifactRoleSidecar, Path: "docs/runbooks/m1-runbook.runbook.yaml", Kind: domain.RunbookSidecarKind, Metadata: sidecarMeta},
			},
		}},
	}
}

func writeRunbookFiles(t *testing.T, sidecar string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m1-runbook.md"), []byte("# m1\n"), 0o600))
	if sidecar != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "m1-runbook.runbook.yaml"), []byte(sidecar), 0o600))
	}
	return root
}

func TestRequireValidRunbookSidecarsAcceptsAParsingConsistentSidecar(t *testing.T) {
	root := writeRunbookFiles(t, runbookSidecarFixture)
	meta := map[string]string{"runbook_id": "m1-runbook", "source_doc": "docs/runbooks/m1-runbook.md"}
	require.NoError(t, requireValidRunbookSidecars(root, runbookSnapshot(meta)))
}

func TestRequireValidRunbookSidecarsRejectsASidecarThatDoesNotParse(t *testing.T) {
	root := writeRunbookFiles(t, "not: a runbook\n")
	meta := map[string]string{"runbook_id": "m1-runbook", "source_doc": "docs/runbooks/m1-runbook.md"}
	require.ErrorContains(t, requireValidRunbookSidecars(root, runbookSnapshot(meta)), "runbook_invalid")
}

func TestRequireValidRunbookSidecarsRejectsAMissingSidecar(t *testing.T) {
	root := writeRunbookFiles(t, "")
	meta := map[string]string{"runbook_id": "m1-runbook", "source_doc": "docs/runbooks/m1-runbook.md"}
	require.ErrorContains(t, requireValidRunbookSidecars(root, runbookSnapshot(meta)), "unreadable")
}

func TestRequireValidRunbookSidecarsRejectsAnIdentityThatDisagreesWithTheSnapshot(t *testing.T) {
	root := writeRunbookFiles(t, runbookSidecarFixture)
	meta := map[string]string{"runbook_id": "another-id", "source_doc": "docs/runbooks/m1-runbook.md"}
	require.ErrorContains(t, requireValidRunbookSidecars(root, runbookSnapshot(meta)), "runbook_id")
}

func TestRequireValidRunbookSidecarsIgnoresSnapshotsWithoutARunbookGroup(t *testing.T) {
	require.NoError(t, requireValidRunbookSidecars(t.TempDir(), domain.ExecutionSnapshot{Version: domain.ExecutionSnapshotVersion, MissionID: "m1"}))
}
