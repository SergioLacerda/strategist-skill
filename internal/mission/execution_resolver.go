package mission

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ExecutionResolveInput is the provider-neutral input to an execution
// resolver. Weapon identity is intentionally absent.
type ExecutionResolveInput struct {
	MissionID   string
	SideQuestID string
	Kind        string
	Origin      string
	Payload     map[string]any
}

// ExecutionResolver converts one accepted intent into normalized execution
// groups. Provider-specific preparation belongs inside the resolver.
type ExecutionResolver interface {
	Key() domain.ResolverKey
	Resolve(context.Context, ExecutionResolveInput) (domain.ExecutionResolution, error)
}

// ResolverRegistry selects resolvers by domain kind and origin.
type ResolverRegistry struct {
	resolvers map[domain.ResolverKey]ExecutionResolver
}

// NewResolverRegistry constructs a registry and rejects duplicate resolver keys.
func NewResolverRegistry(resolvers ...ExecutionResolver) (*ResolverRegistry, error) {
	registry := &ResolverRegistry{resolvers: make(map[domain.ResolverKey]ExecutionResolver, len(resolvers))}
	for _, resolver := range resolvers {
		if err := registry.Register(resolver); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// Register adds one resolver keyed by its domain kind and origin.
func (r *ResolverRegistry) Register(resolver ExecutionResolver) error {
	if resolver == nil {
		return fmt.Errorf("execution_resolver_invalid: resolver is nil")
	}
	key := resolver.Key()
	if err := key.Validate(); err != nil {
		return fmt.Errorf("execution resolver key: %w", err)
	}
	if _, exists := r.resolvers[key]; exists {
		return fmt.Errorf("execution_resolver_duplicate: %s/%s", key.Kind, key.Origin)
	}
	r.resolvers[key] = resolver
	return nil
}

// Has reports whether a resolver is registered for the domain kind and origin.
func (r *ResolverRegistry) Has(kind, origin string) bool {
	_, ok := r.resolvers[domain.ResolverKey{Kind: kind, Origin: origin}]
	return ok
}

// Resolve dispatches one provider-neutral intent to its registered resolver.
func (r *ResolverRegistry) Resolve(ctx context.Context, input ExecutionResolveInput) (domain.ExecutionResolution, error) {
	key := domain.ResolverKey{Kind: input.Kind, Origin: input.Origin}
	resolver, ok := r.resolvers[key]
	if !ok {
		return domain.ExecutionResolution{}, fmt.Errorf("execution_resolver_not_registered: %s/%s", key.Kind, key.Origin)
	}
	resolution, err := resolver.Resolve(ctx, input)
	if err != nil {
		return domain.ExecutionResolution{}, fmt.Errorf("resolve execution intent %s/%s: %w", key.Kind, key.Origin, err)
	}
	if err := resolution.Validate(input.MissionID); err != nil {
		return domain.ExecutionResolution{}, fmt.Errorf("execution_resolver_invalid_output: %w", err)
	}
	return resolution, nil
}

type documentationExecutionResolver struct{}

func (documentationExecutionResolver) Key() domain.ResolverKey {
	return domain.ResolverKey{Kind: domain.ExecutionTargetKindDocumentation, Origin: domain.ExecutionTargetOriginRefinedPackage}
}

func (documentationExecutionResolver) Resolve(_ context.Context, input ExecutionResolveInput) (domain.ExecutionResolution, error) {
	paths, ok := input.Payload["paths"].([]string)
	if !ok {
		return domain.ExecutionResolution{}, fmt.Errorf("execution_resolver_invalid_output: documentation paths are missing")
	}
	group := domain.ExecutionGroup{
		ID: "documentation-" + input.MissionID, Resolver: documentationExecutionResolver{}.Key(),
		Kind: domain.ExecutionTargetKindDocumentation, Origin: domain.ExecutionTargetOriginRefinedPackage,
		CompletionPolicy: domain.ExecutionCompletionAllRequired,
	}
	for index, path := range paths {
		group.Artifacts = append(group.Artifacts, domain.ExecutionArtifact{
			ID: fmt.Sprintf("documentation-%03d", index+1), Role: domain.ExecutionArtifactRolePrimary,
			Path: path, Kind: domain.ExecutionTargetKindDocumentation,
			Metadata: map[string]string{"completion_source": "refined_tasks"},
		})
	}
	return domain.ExecutionResolution{Groups: []domain.ExecutionGroup{group}}, nil
}

type adrExecutionResolver struct {
	strategistRoot string
	basePath       string
	now            timeProvider
}

type timeProvider interface {
	Now() time.Time
}

type fixedTimeProvider struct{ value time.Time }

func (p fixedTimeProvider) Now() time.Time { return p.value }

func (r adrExecutionResolver) Key() domain.ResolverKey {
	return domain.ResolverKey{Kind: domain.ExecutionTargetKindADR, Origin: domain.ExecutionTargetOriginAcceptedSideQuest}
}

func (r adrExecutionResolver) Resolve(_ context.Context, input ExecutionResolveInput) (domain.ExecutionResolution, error) {
	path, err := ResolveADRTarget(ADRTargetRequest{StrategistRoot: r.strategistRoot, BasePath: r.basePath, MissionID: input.MissionID, State: domain.StateExecution, Now: r.now.Now()})
	if err != nil {
		return domain.ExecutionResolution{}, err
	}
	groupID := "adr-" + input.SideQuestID
	return domain.ExecutionResolution{Groups: []domain.ExecutionGroup{{
		ID: groupID, Resolver: r.Key(), SideQuestID: input.SideQuestID, Kind: domain.ExecutionTargetKindADR, Origin: domain.ExecutionTargetOriginAcceptedSideQuest,
		CompletionPolicy: domain.ExecutionCompletionAllRequired,
		Artifacts:        []domain.ExecutionArtifact{{ID: groupID + "-main", Role: domain.ExecutionArtifactRolePrimary, Path: path, Kind: domain.ExecutionTargetKindADR}},
	}}}, nil
}

type runbookExecutionResolver struct {
	strategistRoot string
	now            timeProvider
}

func (r runbookExecutionResolver) Key() domain.ResolverKey {
	return domain.ResolverKey{Kind: "runbook", Origin: domain.ExecutionTargetOriginAcceptedSideQuest}
}

func (r runbookExecutionResolver) Resolve(_ context.Context, input ExecutionResolveInput) (domain.ExecutionResolution, error) {
	rec, found, err := LoadSideQuest(r.strategistRoot, input.MissionID, input.SideQuestID)
	if err != nil {
		return domain.ExecutionResolution{}, err
	}
	if !found || rec.Decision != SideQuestAccepted {
		return domain.ExecutionResolution{}, fmt.Errorf("runbook_side_quest_not_accepted: %s", input.SideQuestID)
	}
	mainPath, err := r.resolveRunbookMainPath(input, rec)
	if err != nil {
		return domain.ExecutionResolution{}, err
	}
	return domain.ExecutionResolution{Groups: []domain.ExecutionGroup{runbookGroup(r.Key(), input.SideQuestID, mainPath)}}, nil
}

func (r runbookExecutionResolver) resolveRunbookMainPath(input ExecutionResolveInput, rec AcceptedSideQuest) (string, error) {
	if rec.ReservedPath != "" {
		return rec.ReservedPath, nil
	}
	mainPath := rec.Destination.Fallback
	if rec.Destination.CanonicalPath != "" {
		mainPath = filepath.ToSlash(filepath.Join(rec.Destination.CanonicalPath, input.MissionID+"-runbook.md"))
	}
	_, err := os.Stat(filepath.Join(filepath.Dir(r.strategistRoot), mainPath))
	if err == nil {
		return "", fmt.Errorf("runbook_target_exists: %s", mainPath)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("runbook_target_unreadable: %w", err)
	}
	return mainPath, nil
}
