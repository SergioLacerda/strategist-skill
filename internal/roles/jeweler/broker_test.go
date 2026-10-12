package jeweler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	readiness Readiness
	query     Projection
	lastQuery QueryRequest
	queries   int
	maintains int
}

func (f *fakeProvider) Query(_ context.Context, request QueryRequest) (Projection, error) {
	f.queries++
	f.lastQuery = request
	return f.query, nil
}
func (f *fakeProvider) Maintain(context.Context, MaintenanceRequest) (MaintenanceResult, error) {
	f.maintains++
	return MaintenanceResult{Operation: "index", Readiness: f.readiness}, nil
}
func (f *fakeProvider) Readiness(context.Context, string) (Readiness, error) { return f.readiness, nil }

func validQuery() QueryRequest {
	return QueryRequest{Requester: "ranger", MissionID: "m-1", CorrelationID: "c-1", Root: "/workspace", Limit: 10}
}

func validIdentity() ProviderIdentity {
	return ProviderIdentity{Package: "treasure-chest", Version: "0.2.0", API: "knowledge-api/v1", Digest: "sha256:provider", Provenance: "https://github.com/sergiolacerda/skill-for-hire@fbd7e8b"}
}

func TestBrokerQueriesOnlyReadyExternalProvider(t *testing.T) {
	provider := &fakeProvider{
		readiness: Readiness{Status: "ready", Reason: "verified"},
		query:     Projection{Identity: validIdentity(), Items: make([]Item, 2)},
	}
	b := Broker{Provider: provider, Now: func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }}

	result, err := b.Query(context.Background(), validQuery())
	require.NoError(t, err)
	require.Equal(t, ContractVersion, result.ContractVersion)
	require.Equal(t, "2026-10-11T12:00:00Z", result.CapturedAt)
	require.Equal(t, "2026-10-11T12:00:00Z", result.Readiness.CheckedAt)
	require.Equal(t, 1, provider.queries)
}

func TestBrokerFailsClosedWhenProviderIsUnavailable(t *testing.T) {
	provider := &fakeProvider{readiness: Readiness{Status: "blocked", Reason: "identity_mismatch"}}
	result, err := (Broker{Provider: provider}).Query(context.Background(), validQuery())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderUnavailable)
	require.Equal(t, "blocked", result.Readiness.Status)
	require.Zero(t, provider.queries)
}

func TestBrokerBoundsConsumerResults(t *testing.T) {
	provider := &fakeProvider{readiness: Readiness{Status: "ready"}, query: Projection{Items: make([]Item, 4)}}
	req := validQuery()
	req.Limit = 2
	result, err := (Broker{Provider: provider}).Query(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Contains(t, result.Limitations, "result_limit_applied")
}

func TestBrokerSelectRunbooksRoutesIntentAndSignals(t *testing.T) {
	provider := &fakeProvider{
		readiness: Readiness{Status: "ready"},
		query:     Projection{Items: []Item{{ID: "runbook-1", Kind: "runbook", Reason: "signal matched"}}},
	}
	req := RunbookSelectionRequest{
		QueryRequest: validQuery(),
		Signals:      []string{"deployment", "timeout"},
	}

	projection, err := (Broker{Provider: provider}).SelectRunbooks(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, projection.Items, 1)
	require.Equal(t, RunbookSelectIntent, provider.lastQuery.Intent)
	require.Equal(t, []string{"deployment", "timeout"}, provider.lastQuery.Concepts)
}

func TestBrokerSelectRunbooksRejectsKnowledgeOnlyProviderResponse(t *testing.T) {
	provider := &fakeProvider{
		readiness: Readiness{Status: "ready"},
		query:     Projection{Items: []Item{{ID: "jewel-1", Kind: "jewel"}}},
	}

	_, err := (Broker{Provider: provider}).SelectRunbooks(context.Background(), RunbookSelectionRequest{
		QueryRequest: validQuery(),
		Signals:      []string{"deployment"},
	})
	require.ErrorContains(t, err, "unsupported item kind")
}

func TestBrokerSelectRunbooksNormalizesPotionIndexEntries(t *testing.T) {
	provider := &fakeProvider{
		readiness: Readiness{Status: "ready"},
		query:     Projection{Items: []Item{{ID: "potion-1", Kind: "potion", Source: Source{Path: "docs/runbooks/timeout.md"}}}},
	}

	projection, err := (Broker{Provider: provider}).SelectRunbooks(context.Background(), RunbookSelectionRequest{
		QueryRequest: validQuery(),
		Signals:      []string{"timeout"},
	})
	require.NoError(t, err)
	require.Equal(t, "runbook", projection.Items[0].Kind)
}

func TestBrokerSelectRunbooksIgnoresMatchingJewels(t *testing.T) {
	provider := &fakeProvider{
		readiness: Readiness{Status: "ready"},
		query: Projection{Items: []Item{
			{ID: "jewel-1", Kind: "jewel", Source: Source{Path: "docs/notes/timeout.md"}},
			{ID: "potion-1", Kind: "potion", Source: Source{Path: "docs/runbooks/timeout.md"}},
		}},
	}

	projection, err := (Broker{Provider: provider}).SelectRunbooks(context.Background(), RunbookSelectionRequest{
		QueryRequest: validQuery(),
		Signals:      []string{"timeout"},
	})
	require.NoError(t, err)
	require.Len(t, projection.Items, 1)
	require.Equal(t, "runbook", projection.Items[0].Kind)
}

func TestRunbookSelectionRequiresNonEmptySignals(t *testing.T) {
	req := RunbookSelectionRequest{QueryRequest: validQuery()}
	require.ErrorContains(t, req.Validate(), "at least one signal")
	req.Signals = []string{"  "}
	require.ErrorContains(t, req.Validate(), "must not be empty")
}

func TestQueryRejectsMutationByConstruction(t *testing.T) {
	req := validQuery()
	req.Requester = "sniper"
	_, err := (Broker{}).Query(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestMaintenanceRequiresExplicitAuthorizationAndIdempotency(t *testing.T) {
	provider := &fakeProvider{readiness: Readiness{Status: "ready"}}
	b := Broker{Provider: provider}
	_, err := b.Maintain(context.Background(), MaintenanceRequest{Operation: "migrate", Root: "/workspace", Principal: "jeweler", CorrelationID: "c", Authorization: "wrong"})
	require.ErrorIs(t, err, ErrUnauthorized)

	_, err = b.Maintain(context.Background(), MaintenanceRequest{Operation: "migrate", Root: "/workspace", Principal: "jeweler", CorrelationID: "c", Authorization: MaintenanceGrant})
	require.Error(t, err)
	require.Contains(t, err.Error(), "idempotency_key")

	_, err = b.Maintain(context.Background(), MaintenanceRequest{Operation: "index", Root: "/workspace", Principal: "jeweler", CorrelationID: "c", Authorization: MaintenanceGrant})
	require.NoError(t, err)
	require.Equal(t, 1, provider.maintains)
}

func TestMaintenanceFailsClosedBeforeCallingUnreadyProvider(t *testing.T) {
	provider := &fakeProvider{readiness: Readiness{Status: "degraded", Reason: "health_failed"}}
	request := MaintenanceRequest{
		Operation: "index", Root: "/workspace", Principal: "jeweler",
		CorrelationID: "c", Authorization: MaintenanceGrant,
	}

	result, err := (Broker{Provider: provider}).Maintain(context.Background(), request)
	require.ErrorIs(t, err, ErrProviderUnavailable)
	require.Equal(t, "health_failed", result.Readiness.Reason)
	require.Zero(t, provider.maintains)
}

func TestCLIProviderDoesNotFallBackWhenBindingIsMissing(t *testing.T) {
	provider := CLIProvider{}
	_, err := provider.Query(context.Background(), validQuery())
	require.Error(t, err)
	require.Contains(t, err.Error(), "executable is not configured")
}

func TestMigrationPlanIsIdempotentAndPreservesConflicts(t *testing.T) {
	manifest := MigrationManifest{
		SchemaVersion: MigrationSchemaVersion, IdempotencyKey: "mig-1",
		Source: validIdentity(), Target: ProviderIdentity{Package: "treasure-chest", Version: "0.2.0", API: "knowledge-api/v1", Digest: "sha256:target", Provenance: "lock"},
		LegacyPreserved: true, RollbackRef: "sha256:legacy",
		Entries: []InventoryEntry{{Key: "jewel-1", Kind: "jewel", SourcePath: "legacy/jewels.yaml", Digest: "sha256:item", Status: "accepted", Provenance: "legacy"}},
	}
	actions, err := PlanMigration(manifest, map[string]string{"jewel-1": "sha256:item"})
	require.NoError(t, err)
	require.Equal(t, []MigrationAction{{Key: "jewel-1", Action: "preserve", Reason: "digest_match"}}, actions)
	actions, err = PlanMigration(manifest, map[string]string{"jewel-1": "sha256:other"})
	require.NoError(t, err)
	require.Equal(t, "conflict", actions[0].Action)
}
