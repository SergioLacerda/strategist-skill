package connectors

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	// InvocationReceiptSchemaVersion identifies the host-issued receipt format.
	InvocationReceiptSchemaVersion = "strategist-invocation-receipt/v1"
	// CapabilityIsolationUnverified means authentication did not prove scoped tools.
	CapabilityIsolationUnverified = "unverified"
	// CapabilityIsolationVerified is reserved for hosts with separate conformance evidence.
	CapabilityIsolationVerified = "verified"
)

// InvocationReceipt is produced by a host only after it invokes the selected
// Weapon. It deliberately excludes prompts, provider output, secrets and home
// directory paths; callers must not persist the nonce outside replay storage.
type InvocationReceipt struct {
	SchemaVersion       string
	MissionID           string
	Role                string
	ProviderID          string
	ResolvedLocation    string
	ResolvedDigest      string
	Nonce               string
	IssuedAt            time.Time
	CapabilityIsolation string
}

// ValidateInvocationReceipt checks the receipt's self-contained shape. Binding
// and replay checks remain Ranger-owned because only Ranger has the mission
// request and nonce store.
func ValidateInvocationReceipt(receipt InvocationReceipt) error {
	if err := validateInvocationReceiptIdentity(receipt); err != nil {
		return err
	}
	return validateInvocationReceiptCompleteness(receipt)
}

// validateInvocationReceiptIdentity checks the schema version and the fields
// that identify who issued the receipt and for what.
func validateInvocationReceiptIdentity(receipt InvocationReceipt) error {
	if receipt.SchemaVersion != InvocationReceiptSchemaVersion {
		return fmt.Errorf("unsupported invocation receipt schema")
	}
	if strings.TrimSpace(receipt.MissionID) == "" || strings.TrimSpace(receipt.Role) == "" || strings.TrimSpace(receipt.ProviderID) == "" {
		return fmt.Errorf("invocation receipt identity is incomplete")
	}
	if strings.TrimSpace(receipt.ResolvedLocation) == "" || filepath.IsAbs(receipt.ResolvedLocation) {
		return fmt.Errorf("invocation receipt location is invalid")
	}
	return nil
}

// validateInvocationReceiptCompleteness checks the remaining self-contained
// fields: the fields a replay/pin check needs, and the declared isolation state.
func validateInvocationReceiptCompleteness(receipt InvocationReceipt) error {
	if strings.TrimSpace(receipt.ResolvedDigest) == "" || strings.TrimSpace(receipt.Nonce) == "" || receipt.IssuedAt.IsZero() {
		return fmt.Errorf("invocation receipt is incomplete")
	}
	switch receipt.CapabilityIsolation {
	case CapabilityIsolationUnverified, CapabilityIsolationVerified:
		return nil
	default:
		return fmt.Errorf("invocation receipt capability isolation is invalid")
	}
}
