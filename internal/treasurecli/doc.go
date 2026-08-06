// Package treasurecli wires the strategist CLI's treasure-chest and runbook
// commands (cobra command trees, flag parsing, table/JSON rendering) onto a
// root command supplied via Register. It is a thin adapter layer over
// internal/treasure's domain logic and internal/runbook's selection logic —
// business logic and YAML/JSON parsing belong in those packages, not here
// (enforced by this package's own TestCmdThinness).
package treasurecli
