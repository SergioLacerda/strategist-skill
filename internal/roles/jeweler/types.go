// Package jeweler owns the Strategist-side support boundary for Treasure
// Chest. It contains no Treasure Chest domain implementation: the external
// provider owns that state, while this package owns authorization, bounded
// envelopes, provenance, and fail-closed routing.
package jeweler

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	// RoleID identifies the JEWELER host role.
	RoleID = "jeweler"
	// CompatibilityAlias identifies the historical JEWELER role alias.
	CompatibilityAlias = "jewelcrafter"
	// SupportSlot identifies the Treasure support capability served by JEWELER.
	SupportSlot = "treasure"
	// ContractVersion identifies the JEWELER boundary contract.
	ContractVersion = "jeweler-boundary/v1"
	// KnowledgeAPIVersion identifies the provider knowledge API version.
	KnowledgeAPIVersion = 1
	// MaintenanceGrant identifies the authorization grant for maintenance.
	MaintenanceGrant = "jeweler:maintenance/v1"
)

var (
	// ErrProviderUnavailable indicates that no admitted provider can answer.
	ErrProviderUnavailable = errors.New("jeweler: external Treasure Chest provider unavailable")
	// ErrUnauthorized indicates that a maintenance grant is missing or invalid.
	ErrUnauthorized = errors.New("jeweler: maintenance authorization required")
	// ErrMutationForbidden indicates that a consumer query attempted mutation.
	ErrMutationForbidden = errors.New("jeweler: consumer query is read-only")
)

// ProviderIdentity is immutable evidence for the provider that answered.
type ProviderIdentity struct {
	Package    string `json:"package"`
	Version    string `json:"version"`
	API        string `json:"api"`
	Digest     string `json:"digest"`
	Provenance string `json:"provenance"`
}

// Validate checks that all immutable provider identity evidence is present.
func (i ProviderIdentity) Validate() error {
	for name, value := range map[string]string{
		"package": i.Package, "version": i.Version, "api": i.API,
		"digest": i.Digest, "provenance": i.Provenance,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("jeweler: provider identity %s is required", name)
		}
	}
	return nil
}

// Freshness describes the age confidence of provider data.
type Freshness string

const (
	// FreshnessFresh indicates current provider data.
	FreshnessFresh Freshness = "fresh"
	// FreshnessStale indicates provider data outside the requested freshness bound.
	FreshnessStale Freshness = "stale"
	// FreshnessUnknown indicates that provider data age cannot be established.
	FreshnessUnknown Freshness = "unknown"
)

// Readiness is the provider activation state and its supporting evidence.
type Readiness struct {
	Status    string         `json:"status"`
	Reason    string         `json:"reason"`
	Detail    string         `json:"detail,omitempty"`
	CheckedAt string         `json:"checked_at"`
	Evidence  EvidenceVector `json:"evidence,omitempty"`
}

// Ready reports whether all provider activation prerequisites passed.
func (r Readiness) Ready() bool { return r.Status == "ready" }

// Item is one provider result with source evidence.
type Item struct {
	ID            string  `json:"ID"`
	Kind          string  `json:"Kind"`
	Score         float64 `json:"Score"`
	Trust         float64 `json:"Trust"`
	Freshness     string  `json:"Freshness"`
	Applicability string  `json:"Applicability"`
	Reason        string  `json:"Reason"`
	Source        Source  `json:"Source"`
}

// Source identifies the evidence location for an item.
type Source struct {
	Path   string `json:"Path"`
	Digest string `json:"Digest,omitempty"`
	Commit string `json:"Commit,omitempty"`
}

// Projection is the bounded provider result returned through JEWELER.
type Projection struct {
	ContractVersion string           `json:"contract_version"`
	Identity        ProviderIdentity `json:"identity"`
	CapturedAt      string           `json:"captured_at"`
	Freshness       Freshness        `json:"freshness"`
	Readiness       Readiness        `json:"readiness"`
	Items           []Item           `json:"items"`
	Limitations     []string         `json:"limitations,omitempty"`
	AliasUsed       string           `json:"alias_used,omitempty"`
}

// MaintenanceRequest carries an explicitly authorized provider operation.
type MaintenanceRequest struct {
	Principal      string `json:"principal"`
	Authorization  string `json:"authorization"`
	Operation      string `json:"operation"`
	Root           string `json:"root"`
	CorrelationID  string `json:"correlation_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// Validate checks authorization, operation, and idempotency requirements.
func (r MaintenanceRequest) Validate() error {
	if r.Authorization != MaintenanceGrant {
		return ErrUnauthorized
	}
	if strings.TrimSpace(r.Principal) == "" || strings.TrimSpace(r.CorrelationID) == "" || strings.TrimSpace(r.Root) == "" {
		return fmt.Errorf("jeweler: maintenance principal, correlation_id, and root are required")
	}
	switch r.Operation {
	case "index", "curate", "migrate", "reconcile", "probe", "health":
	default:
		return fmt.Errorf("jeweler: unsupported maintenance operation %q", r.Operation)
	}
	if (r.Operation == "migrate" || r.Operation == "reconcile") && strings.TrimSpace(r.IdempotencyKey) == "" {
		return fmt.Errorf("jeweler: %s requires idempotency_key", r.Operation)
	}
	return nil
}

// MaintenanceResult reports the outcome and readiness after maintenance.
type MaintenanceResult struct {
	Operation string    `json:"operation"`
	Readiness Readiness `json:"readiness"`
	Detail    string    `json:"detail,omitempty"`
}

// Provider is the only implementation seam. Production uses the external
// provider adapter; tests may inject a deterministic fixture explicitly.
type Provider interface {
	Query(context.Context, QueryRequest) (Projection, error)
	Maintain(context.Context, MaintenanceRequest) (MaintenanceResult, error)
	Readiness(context.Context, string) (Readiness, error)
}
