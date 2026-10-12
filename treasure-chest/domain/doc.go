// Package domain holds the treasure-chest-specific domain vocabulary —
// chest grading, jewel and potion lifecycle statuses, and promotion packets
// — isolated from the shared internal/domain package by the
// 20260915-treasure-chest-relocation-verification mission (see
// docs/adr/0040-treasure-chest-in-repo-isolation-staging.md). It stays a
// separate package from internal/domain deliberately: treasure-chest code
// should depend on this narrow surface, not the whole shared domain
// package. Evidence and confidence values used by jewel evidence fields are
// defined here as part of the provider's host-neutral interchange contract.
package domain
