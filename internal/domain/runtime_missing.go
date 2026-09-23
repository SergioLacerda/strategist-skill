package domain

import "fmt"

// ReasonRuntimeMissing is the catalog token (contracts/machine/errors.yaml) `strategist
// check` reports when a required runtime file is absent.
const ReasonRuntimeMissing = "runtime_missing"

// GeneratedRuntimeFilePaths returns runtime files that `strategist compile`
// generates rather than installs (so they have no embedded default to compare
// bytes against) but that an installed runtime must still contain: the Strategist
// skill entrypoint reads agent-protocol.md at startup.
func GeneratedRuntimeFilePaths() []string {
	return []string{"agent-protocol.md"}
}

// FormatRuntimeMissingDiagnostic formats the check diagnostic for a Required
// normative default file that is absent from the runtime.
func FormatRuntimeMissingDiagnostic(path string) string {
	return fmt.Sprintf("%s: normative file %q is missing — run strategist install", ReasonRuntimeMissing, path)
}

// FormatGeneratedRuntimeMissingDiagnostic formats the check diagnostic for a
// generated runtime file that is absent.
func FormatGeneratedRuntimeMissingDiagnostic(path string) string {
	return fmt.Sprintf("%s: generated file %q is missing — run strategist compile", ReasonRuntimeMissing, path)
}
