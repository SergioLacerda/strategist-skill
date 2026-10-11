package mission

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

type testExecutionResolver struct {
	key domain.ResolverKey
}

func (r testExecutionResolver) Key() domain.ResolverKey { return r.key }

func (r testExecutionResolver) Resolve(_ context.Context, _ ExecutionResolveInput) (domain.ExecutionResolution, error) {
	group := domain.ExecutionGroup{ID: "group-1", Resolver: r.key, Kind: r.key.Kind, Origin: r.key.Origin, CompletionPolicy: domain.ExecutionCompletionAllRequired,
		Artifacts: []domain.ExecutionArtifact{{ID: "artifact-1", Role: domain.ExecutionArtifactRolePrimary, Path: "docs/one.md", Kind: r.key.Kind}}}
	return domain.ExecutionResolution{Groups: []domain.ExecutionGroup{group}}, nil
}

func TestResolverRegistryUsesKindAndOriginAndRejectsDuplicate(t *testing.T) {
	resolver := testExecutionResolver{key: domain.ResolverKey{Kind: "future", Origin: "accepted_intent"}}
	registry, err := NewResolverRegistry(resolver)
	require.NoError(t, err)
	_, err = registry.Resolve(context.Background(), ExecutionResolveInput{MissionID: "m1", Kind: resolver.key.Kind, Origin: resolver.key.Origin})
	require.NoError(t, err)
	require.ErrorContains(t, registry.Register(resolver), "execution_resolver_duplicate")
	_, err = registry.Resolve(context.Background(), ExecutionResolveInput{MissionID: "m1", Kind: "missing", Origin: resolver.key.Origin})
	require.ErrorContains(t, err, "execution_resolver_not_registered")
}

func TestBuiltinRunbookResolverProducesPrimaryAndSidecar(t *testing.T) {
	f := newSQFixture(t, "m-runbook")
	path := filepath.Join(f.base, "refined", "m-runbook", "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("# t\n```yaml\nside_quests_approved:\n  - {id: OA-RUNBOOK-m-runbook, description: procedure, strategy: execute_later, status: proposed}\n```\n"), 0o600))
	_, err := DecideSideQuest(SideQuestDecision{StrategistRoot: f.root, BasePath: f.base, MissionID: "m-runbook", SideQuestID: OARunbookSideQuestID("m-runbook"), Decision: SideQuestAccepted, State: domain.StateApprovalGate, CanonicalPath: "docs/runbooks"})
	require.NoError(t, err)
	snapshot, err := ResolveExecutionSnapshot(f.root, f.base, "m-runbook", nil, time.Unix(100, 0))
	require.NoError(t, err)
	require.Len(t, snapshot.Groups, 1)
	require.Len(t, snapshot.Groups[0].Artifacts, 2)
	require.Equal(t, domain.ExecutionArtifactRoleSidecar, snapshot.Groups[0].Artifacts[1].Role)
}
