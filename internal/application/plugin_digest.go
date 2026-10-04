package application

import (
	"errors"
	"fmt"
	"strings"
)

// ResolvedDigestStatus is the provider-neutral result of comparing a resolved
// Weapon digest with its catalog pin.
type ResolvedDigestStatus string

const (
	// ResolvedDigestMatch indicates that the resolved digest matches the catalog pin.
	ResolvedDigestMatch ResolvedDigestStatus = "match"
	// ResolvedDigestMismatch indicates that the resolved digest differs from the catalog pin.
	ResolvedDigestMismatch ResolvedDigestStatus = "mismatch"
	// ResolvedDigestPinUnavailable indicates that the catalog has no usable pin.
	ResolvedDigestPinUnavailable ResolvedDigestStatus = "pin_unavailable"
	// ResolvedDigestUnknown indicates that the comparison has no conclusive result.
	ResolvedDigestUnknown ResolvedDigestStatus = "unknown"
)

// ResolvedDigestComparison is the application DTO rendered by the CLI.
type ResolvedDigestComparison struct {
	Status   ResolvedDigestStatus
	Pin      string
	Resolved string
}

// ResolveDigestInput chooses a caller-supplied digest or delegates file
// hashing to the filesystem adapter. The two input modes are exclusive.
func ResolveDigestInput(digest, file string, hashFile func(string) (string, error)) (string, error) {
	hasDigest, hasFile := strings.TrimSpace(digest) != "", strings.TrimSpace(file) != ""
	switch {
	case hasDigest && hasFile:
		return "", errors.New("--resolved-digest and --file are mutually exclusive")
	case hasFile:
		if hashFile == nil {
			return "", errors.New("resolved-digest: file hash adapter is required")
		}
		resolved, err := hashFile(file)
		if err != nil {
			return "", fmt.Errorf("resolved-digest: %w", err)
		}
		return resolved, nil
	default:
		return digest, nil
	}
}

// CompareResolvedDigest applies the fail-closed mismatch policy while the
// catalog connector remains injected at the composition boundary.
func CompareResolvedDigest(provider, resolved string, compare func(provider, resolved string) (ResolvedDigestComparison, error)) (ResolvedDigestComparison, error) {
	if compare == nil {
		return ResolvedDigestComparison{}, errors.New("resolved-digest: catalog comparison adapter is required")
	}
	return compare(provider, resolved)
}
