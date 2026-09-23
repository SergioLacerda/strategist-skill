//go:build !windows

package runtimeenv

import (
	"os/exec"
	"syscall"
)

// boundToProcessGroup puts cmd in its own process group and, on context
// cancellation, kills that whole group instead of only the direct child.
//
// exec.CommandContext's default Cancel only signals cmd.Process. When the
// launched executable is a shell script, the shell can fork a grandchild
// (e.g. `sleep`) that outlives the killed shell and keeps the inherited
// stdout/stderr pipes open, so cmd.Wait()/cmd.CombinedOutput() blocks until
// that orphan exits on its own — the context deadline stops bounding
// wall-clock time it was meant to bound. Setpgid makes the child its own
// group leader (pgid == child pid), so -pid reaches every process it forked.
func boundToProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
