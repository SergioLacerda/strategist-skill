package jeweler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// InventoryEntry is a provenance-preserving snapshot of one legacy record.
type InventoryEntry struct {
	Key        string `json:"key"`
	Kind       string `json:"kind"`
	SourcePath string `json:"source_path"`
	Digest     string `json:"digest"`
	Status     string `json:"status"`
	Provenance string `json:"provenance"`
}

// MigrationManifest is immutable evidence for a Treasure Chest cutover. It
// describes a transfer but deliberately does not perform one.
type MigrationManifest struct {
	SchemaVersion   string           `json:"schema_version"`
	IdempotencyKey  string           `json:"idempotency_key"`
	Source          ProviderIdentity `json:"source"`
	Target          ProviderIdentity `json:"target"`
	Entries         []InventoryEntry `json:"entries"`
	LegacyPreserved bool             `json:"legacy_preserved"`
	RollbackRef     string           `json:"rollback_ref"`
}

// MigrationSchemaVersion identifies the immutable migration manifest schema.
const MigrationSchemaVersion = "jeweler-migration/v1"

var (
	// ErrMigrationApprovalRequired indicates that an execution lacks explicit approval.
	ErrMigrationApprovalRequired = errors.New("jeweler: migration approval required")
	// ErrMigrationApplierUnavailable indicates that no migration applier is bound.
	ErrMigrationApplierUnavailable = errors.New("jeweler: migration applier unavailable")
	// ErrMigrationConflict indicates that a destination digest differs from the manifest.
	ErrMigrationConflict = errors.New("jeweler: migration contains destination conflicts")
	// ErrRollbackApplierUnavailable indicates that no rollback applier is bound.
	ErrRollbackApplierUnavailable = errors.New("jeweler: rollback applier unavailable")
)

// Validate checks the migration manifest before it is planned or executed.
func (m MigrationManifest) Validate() error {
	if err := validateMigrationHeader(m); err != nil {
		return err
	}
	if err := m.Source.Validate(); err != nil {
		return fmt.Errorf("jeweler: migration source: %w", err)
	}
	if err := m.Target.Validate(); err != nil {
		return fmt.Errorf("jeweler: migration target: %w", err)
	}
	if !m.LegacyPreserved {
		return fmt.Errorf("jeweler: migration must preserve legacy data")
	}
	return validateMigrationEntries(m.Entries)
}

func validateMigrationHeader(m MigrationManifest) error {
	if m.SchemaVersion != MigrationSchemaVersion {
		return fmt.Errorf("jeweler: migration schema %q is unsupported", m.SchemaVersion)
	}
	if strings.TrimSpace(m.IdempotencyKey) == "" || strings.TrimSpace(m.RollbackRef) == "" {
		return fmt.Errorf("jeweler: migration idempotency_key and rollback_ref are required")
	}
	return nil
}

func validateMigrationEntries(entries []InventoryEntry) error {
	return validateEntries(entries, "migration")
}

// CanonicalJSON returns the validated manifest in a stable key order. Entry
// order is not an ownership signal, so canonicalization sorts it by key before
// hashing or persisting a replay reference.
func (m MigrationManifest) CanonicalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	canonical := m
	canonical.Entries = append([]InventoryEntry(nil), m.Entries...)
	sort.Slice(canonical.Entries, func(i, j int) bool {
		return canonical.Entries[i].Key < canonical.Entries[j].Key
	})
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("jeweler: marshal migration manifest: %w", err)
	}
	return raw, nil
}

// Digest returns the stable identity of a validated migration manifest.
func (m MigrationManifest) Digest() (string, error) {
	raw, err := m.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
