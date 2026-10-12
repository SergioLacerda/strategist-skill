package jeweler

import (
	"context"
	"fmt"
	"time"
)

// Broker is the sole host-owned access boundary for Treasure Chest.
type Broker struct {
	Provider Provider
	Now      func() time.Time
}

func (b Broker) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

// Query validates and executes a bounded, read-only provider query.
func (b Broker) Query(ctx context.Context, req QueryRequest) (Projection, error) {
	if err := req.Validate(); err != nil {
		return Projection{}, err
	}
	readiness, err := b.providerReadiness(ctx, req.Root)
	if err != nil {
		return Projection{Readiness: readiness}, fmt.Errorf("jeweler: readiness: %w", err)
	}
	if !readiness.Ready() {
		return Projection{Readiness: readiness}, fmt.Errorf("%w: %s", ErrProviderUnavailable, readiness.Reason)
	}
	projection, err := b.Provider.Query(ctx, req)
	if err != nil {
		return Projection{}, fmt.Errorf("jeweler: query: %w", err)
	}
	return b.completeProjection(projection, readiness, req.Limit), nil
}

func (b Broker) providerReadiness(ctx context.Context, root string) (Readiness, error) {
	if b.Provider == nil {
		return Readiness{Status: "unavailable", Reason: "provider_not_bound"}, ErrProviderUnavailable
	}
	readiness, err := b.Provider.Readiness(ctx, root)
	if readiness.CheckedAt == "" {
		readiness.CheckedAt = b.now().Format(time.RFC3339)
	}
	if err != nil {
		return readiness, fmt.Errorf("provider readiness: %w", err)
	}
	return readiness, nil
}

func (b Broker) completeProjection(projection Projection, readiness Readiness, limit int) Projection {
	if projection.ContractVersion == "" {
		projection.ContractVersion = ContractVersion
	}
	if projection.CapturedAt == "" {
		projection.CapturedAt = b.now().Format(time.RFC3339)
	}
	if projection.Readiness.Status == "" {
		projection.Readiness = readiness
	} else if projection.Readiness.CheckedAt == "" {
		projection.Readiness.CheckedAt = readiness.CheckedAt
	}
	if len(projection.Items) > limit {
		projection.Items = projection.Items[:limit]
		projection.Limitations = append(projection.Limitations, "result_limit_applied")
	}
	return projection
}

// Maintain validates and executes an authorized provider maintenance operation.
func (b Broker) Maintain(ctx context.Context, req MaintenanceRequest) (MaintenanceResult, error) {
	if err := req.Validate(); err != nil {
		return MaintenanceResult{}, err
	}
	readiness, err := b.providerReadiness(ctx, req.Root)
	if err != nil {
		return MaintenanceResult{Operation: req.Operation, Readiness: readiness}, fmt.Errorf("jeweler: maintenance readiness: %w", err)
	}
	if !readiness.Ready() {
		return MaintenanceResult{Operation: req.Operation, Readiness: readiness}, fmt.Errorf("%w: %s", ErrProviderUnavailable, readiness.Reason)
	}
	result, err := b.Provider.Maintain(ctx, req)
	if result.Readiness.CheckedAt == "" {
		result.Readiness.CheckedAt = b.now().Format(time.RFC3339)
	}
	if result.Readiness.Status == "" {
		result.Readiness = readiness
	}
	if err != nil {
		return result, fmt.Errorf("jeweler: maintenance %s: %w", req.Operation, err)
	}
	return result, nil
}
