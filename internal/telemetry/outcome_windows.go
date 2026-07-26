//go:build windows

package telemetry

import "os"

// Windows builds currently run without an advisory inter-process lock. The
// outcome writer still validates and deduplicates records inside the process,
// but concurrent processes do not get the Unix flock guarantee.
func lockFile(_ *os.File) error   { return nil }
func unlockFile(_ *os.File) error { return nil }
