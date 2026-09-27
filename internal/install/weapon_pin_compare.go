package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// ResolvedDigestStatus is the outcome of comparing a self-reported
// weapon_invocation.resolved_digest (contracts/narrative/03-discovery.md §
// Weapon Profile for a Delegated Ranger) against the catalog's
// upstream_content_digest pin. This is a read-only report: it never
// invokes anything and never treats an unavailable pin as a match.
type ResolvedDigestStatus string

const (
	// ResolvedDigestMatch means the resolved bytes hash to the pinned digest.
	ResolvedDigestMatch ResolvedDigestStatus = "match"
	// ResolvedDigestMismatch means both values are present and differ.
	ResolvedDigestMismatch ResolvedDigestStatus = "mismatch"
	// ResolvedDigestPinUnavailable means the provider declares no
	// upstream_content_digest to compare against (never reported as a match).
	ResolvedDigestPinUnavailable ResolvedDigestStatus = "pin_unavailable"
	// ResolvedDigestNotReported means the caller supplied no resolved digest
	// to compare (a Ranger run that did not fill weapon_invocation.resolved_digest).
	ResolvedDigestNotReported ResolvedDigestStatus = "not_reported"
)

// ResolvedDigestComparison is CompareResolvedDigest's result.
type ResolvedDigestComparison struct {
	Status   ResolvedDigestStatus
	Pin      string
	Resolved string
}

// CompareResolvedDigest reads catalogPath (plain filesystem, maintainer/CI
// path) for providerID's upstream_content_digest pin and compares it with
// resolvedDigest, a value the caller already has (typically read from a
// Ranger handoff's weapon_invocation.resolved_digest field). It never
// invokes the Weapon and never computes resolvedDigest itself — see
// HashFileSHA256 for that, kept separate so a caller with only a digest in
// hand (no file) can still compare.
func CompareResolvedDigest(catalogPath, providerID, resolvedDigest string) (ResolvedDigestComparison, error) {
	pin, found, err := ReadUpstreamContentDigest(catalogPath, providerID)
	if err != nil {
		return ResolvedDigestComparison{}, err
	}
	result := ResolvedDigestComparison{Pin: pin, Resolved: resolvedDigest}
	switch {
	case !found:
		result.Status = ResolvedDigestPinUnavailable
	case resolvedDigest == "":
		result.Status = ResolvedDigestNotReported
	case pin == resolvedDigest:
		result.Status = ResolvedDigestMatch
	default:
		result.Status = ResolvedDigestMismatch
	}
	return result, nil
}

// HashFileSHA256 returns "sha256:<64 hex>" of path's raw bytes — the format
// contracts/narrative/03-discovery.md's weapon_invocation.resolved_digest
// declares. It is a plain read; it never writes and never invokes anything.
func HashFileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: caller-supplied path to the Weapon file it already resolved and read
	if err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
