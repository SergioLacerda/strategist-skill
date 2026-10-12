// Package treasurecli wires the strategist CLI's treasure-chest and runbook
// commands (cobra command trees, flag parsing, table/JSON rendering) onto a
// root command supplied via Register. Production runbook selection is a thin
// adapter over JEWELER; the deprecated local selector remains isolated for
// staging tests. Business logic and YAML/JSON parsing belong in their owning
// packages, not here (enforced by this package's own TestCmdThinness).
package treasurecli
