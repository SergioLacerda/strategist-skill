package main

import "github.com/spf13/cobra"

// pendingDisplayRestore undoes the running command's quiet display level. The
// CLI runs one command per process, so a single slot is enough; it exists only
// so the level is restored after the mission metrics line is emitted (post-run)
// and also when the command fails (cobra finalizers run on both paths), which
// keeps in-process callers such as tests from inheriting a lowered level.
var pendingDisplayRestore func()

func setDisplayRestore(restore func()) { pendingDisplayRestore = restore }

func restoreDisplayLevel() {
	if pendingDisplayRestore != nil {
		pendingDisplayRestore()
		pendingDisplayRestore = nil
	}
}

func init() { cobra.OnFinalize(restoreDisplayLevel) }
