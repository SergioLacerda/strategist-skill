package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionSnapshotRoundTripSupportsMultipleGroupsAndSidecar(t *testing.T) {
	snapshot := ExecutionSnapshot{
		Version: ExecutionSnapshotVersion, MissionID: "m1",
		Groups: []ExecutionGroup{
			{ID: "runbook-1", Resolver: ResolverKey{Kind: "runbook", Origin: ExecutionTargetOriginAcceptedSideQuest}, SideQuestID: "OA-RUNBOOK-m1-1", Kind: "runbook", Origin: ExecutionTargetOriginAcceptedSideQuest, CompletionPolicy: ExecutionCompletionAllRequired,
				Artifacts: []ExecutionArtifact{{ID: "runbook-1-main", Role: ExecutionArtifactRolePrimary, Path: "docs/runbooks/one.md", Kind: "runbook"}, {ID: "runbook-1-sidecar", Role: ExecutionArtifactRoleSidecar, Path: "docs/runbooks/one.runbook.yaml", Kind: RunbookSidecarKind}}},
			{ID: "adr-1", Resolver: ResolverKey{Kind: ExecutionTargetKindADR, Origin: ExecutionTargetOriginAcceptedSideQuest}, SideQuestID: "OA-ADR-m1", Kind: ExecutionTargetKindADR, Origin: ExecutionTargetOriginAcceptedSideQuest, CompletionPolicy: ExecutionCompletionAllRequired,
				Artifacts: []ExecutionArtifact{{ID: "adr-1-main", Role: ExecutionArtifactRolePrimary, Path: "docs/adr/0001.md", Kind: ExecutionTargetKindADR}}},
		},
	}
	require.NoError(t, snapshot.Validate())
	raw, err := json.Marshal(snapshot)
	require.NoError(t, err)
	var got ExecutionSnapshot
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, snapshot, got)
}

func TestExecutionGroupRejectsDuplicateArtifactsAndInvalidPolicy(t *testing.T) {
	group := ExecutionGroup{ID: "g", Resolver: ResolverKey{Kind: "runbook", Origin: "accepted_side_quest"}, Kind: "runbook", Origin: "accepted_side_quest", CompletionPolicy: ExecutionCompletionAllRequired,
		Artifacts: []ExecutionArtifact{{ID: "a", Role: ExecutionArtifactRolePrimary, Path: "docs/a.md", Kind: "runbook"}, {ID: "a", Role: ExecutionArtifactRoleSidecar, Path: "docs/a.json", Kind: "metadata"}}}
	require.ErrorContains(t, group.Validate(), "duplicate artifact id")
	group.Artifacts = group.Artifacts[:1]
	group.CompletionPolicy = "any"
	require.ErrorContains(t, group.Validate(), "unsupported completion policy")
}

func TestSnapshotFromTargetsCreatesOneArtifactGroupsWithoutExpansion(t *testing.T) {
	targets := []ExecutionTarget{{Path: "docs/guide.md", Kind: ExecutionTargetKindDocumentation, Origin: ExecutionTargetOriginRefinedPackage}}
	snapshot, err := SnapshotFromTargets("m1", targets)
	require.NoError(t, err)
	require.Len(t, snapshot.Groups, 1)
	require.Equal(t, "docs/guide.md", snapshot.Groups[0].Artifacts[0].Path)
	require.Equal(t, []string{"docs/guide.md"}, snapshot.ArtifactPaths())
}
