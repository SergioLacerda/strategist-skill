package provider

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

type discoveryReceiptVerification struct {
	authenticated string
	pinStatus     string
	isolation     string
}

func failedReceiptVerification() discoveryReceiptVerification {
	return discoveryReceiptVerification{authenticated: "failed", pinStatus: "not_checked", isolation: "unverified"}
}

func validateInvocationReceipt(request DiscoveryWeaponRequest, receipt connectors.InvocationReceipt) (discoveryReceiptVerification, error) {
	verification := failedReceiptVerification()
	if err := authenticateInvocationReceipt(request, receipt); err != nil {
		return verification, err
	}
	verification.authenticated = "authenticated"
	verification.isolation = receipt.CapabilityIsolation
	if err := verifyCapabilityIsolationClaim(request, receipt); err != nil {
		return failedReceiptVerification(), err
	}
	return verifyReceiptPin(request, receipt, verification)
}

// authenticateInvocationReceipt validates the receipt's shape, binds it to
// this request's identity, rejects a stale receipt, and claims its nonce
// once (replay protection).
func authenticateInvocationReceipt(request DiscoveryWeaponRequest, receipt connectors.InvocationReceipt) error {
	if err := connectors.ValidateInvocationReceipt(receipt); err != nil {
		return fmt.Errorf("validate invocation receipt: %w", err)
	}
	if receipt.MissionID != request.MissionID || receipt.Role != request.Role || receipt.ProviderID != request.ProviderID {
		return fmt.Errorf("invocation receipt identity mismatch")
	}
	now := time.Now()
	if now.Sub(receipt.IssuedAt) > 5*time.Minute || receipt.IssuedAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("invocation receipt is stale")
	}
	if request.ReceiptStore == nil {
		return fmt.Errorf("receipt nonce storage is unavailable")
	}
	if err := request.ReceiptStore.Claim(request.MissionID, receipt.Nonce, receipt.IssuedAt); err != nil {
		return fmt.Errorf("claim invocation receipt nonce: %w", err)
	}
	return nil
}

// verifyCapabilityIsolationClaim rejects a "verified" claim the connector did
// not independently assert, so a receipt alone can never prove isolation.
func verifyCapabilityIsolationClaim(request DiscoveryWeaponRequest, receipt connectors.InvocationReceipt) error {
	if receipt.CapabilityIsolation == connectors.CapabilityIsolationVerified && !request.CapabilityIsolationVerified {
		return fmt.Errorf("capability isolation is not host-verified")
	}
	return nil
}

// verifyReceiptPin compares the receipt's resolved digest against the
// declared catalog pin when one is configured, reporting an explicit
// "pin_unavailable" status rather than silently skipping the check.
func verifyReceiptPin(request DiscoveryWeaponRequest, receipt connectors.InvocationReceipt, verification discoveryReceiptVerification) (discoveryReceiptVerification, error) {
	if request.CatalogPath == "" {
		verification.pinStatus = "pin_unavailable"
		return verification, nil
	}
	comparison, err := install.CompareResolvedDigest(request.CatalogPath, request.ProviderID, receipt.ResolvedDigest)
	if err != nil {
		return failedReceiptVerification(), fmt.Errorf("compare invocation receipt digest: %w", err)
	}
	verification.pinStatus = string(comparison.Status)
	if comparison.Status == install.ResolvedDigestMismatch || comparison.Status == install.ResolvedDigestNotReported {
		return verification, fmt.Errorf("invocation receipt digest does not match declared pin")
	}
	return verification, nil
}

func validateEmbeddedInvocationReceipt(request DiscoveryWeaponRequest, receipt connectors.EmbeddedInvocationReceipt) error {
	if err := receipt.Validate(); err != nil {
		return fmt.Errorf("validate embedded invocation receipt: %w", err)
	}
	if receipt.MissionID != request.MissionID || receipt.Role != request.Role || receipt.Slot != request.Slot || receipt.WeaponID != request.ProviderID {
		return fmt.Errorf("embedded invocation receipt identity mismatch")
	}
	return validateEmbeddedReceiptEvidence(request, receipt)
}

// validateEmbeddedReceiptEvidence checks the digests and the correlation nonce
// the request pinned; empty request values are not constrained.
func validateEmbeddedReceiptEvidence(request DiscoveryWeaponRequest, receipt connectors.EmbeddedInvocationReceipt) error {
	if request.BindingDigest != "" && receipt.BindingDigest != request.BindingDigest {
		return fmt.Errorf("embedded invocation receipt binding digest mismatch")
	}
	if request.SourceDigest != "" && receipt.SourceDigest != request.SourceDigest {
		return fmt.Errorf("embedded invocation receipt source digest mismatch")
	}
	if request.InvocationNonce != "" && receipt.Nonce != request.InvocationNonce {
		return fmt.Errorf("embedded invocation receipt nonce mismatch")
	}
	return validateEmbeddedReceiptAdapter(request, receipt)
}

// validateEmbeddedReceiptAdapter binds the receipt's adapter provenance to the
// mode Strategist committed for the request. Constrains nothing when the
// request pins no adapter.
func validateEmbeddedReceiptAdapter(request DiscoveryWeaponRequest, receipt connectors.EmbeddedInvocationReceipt) error {
	if request.InvocationRequestID != "" && receipt.RequestID != request.InvocationRequestID {
		return fmt.Errorf("embedded invocation receipt request id mismatch")
	}
	if request.ExecutionAdapter == "" {
		return nil
	}
	if receipt.ExecutionAdapter != request.ExecutionAdapter || receipt.ChildPolicyID != request.ChildPolicyID {
		return fmt.Errorf("embedded invocation receipt execution adapter mismatch")
	}
	return nil
}
