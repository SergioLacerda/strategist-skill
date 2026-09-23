//go:build windows

package runtimeenv

import "os/exec"

// boundToProcessGroup is a no-op on Windows. Bounding wall-clock time for a
// killed process's descendants there needs a Job Object, a materially larger
// change than the POSIX process-group fix; left as a known gap (see
// command_unix.go for the Unix behavior this intentionally does not match).
func boundToProcessGroup(_ *exec.Cmd) {}
