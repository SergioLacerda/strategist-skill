// Package domain holds the treasure-chest-specific domain vocabulary —
// chest grading, jewel and potion lifecycle statuses, and promotion packets
// — isolated from the shared internal/domain package by the
// 20260915-treasure-chest-relocation-verification mission (see
// docs/adr/0040-treasure-chest-in-repo-isolation-staging.md). It stays a
// separate package from internal/domain deliberately: treasure-chest code
// should depend on this narrow surface, not the whole shared domain
// package, and internal/domain's Evidence/Confidence vocabulary (used by
// jewel evidence fields) remains genuinely shared and is imported from
// there directly where needed.
package domain
