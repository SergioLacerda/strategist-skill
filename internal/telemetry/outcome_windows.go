//go:build windows

package telemetry

import "os"

func lockFile(_ *os.File) error   { return nil }
func unlockFile(_ *os.File) error { return nil }
