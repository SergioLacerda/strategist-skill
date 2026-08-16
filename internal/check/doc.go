// Package check wires the strategist CLI's `check` and `check-stale`
// commands (cobra command trees, flag parsing, slot/persona/runtime
// validation) onto a root command supplied via Register. Extracted from
// cmd/strategist (20260816-cmd-strategist-cli-reorg), mirroring the
// internal/treasurecli precedent (20260806-treasure-chest-cmd-consolidation)
// — unlike treasurecli, this package owns its validation logic directly
// rather than wrapping a separate domain package, since none existed for
// `check` before this extraction.
package check
