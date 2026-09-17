package treasurecli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/integrity"
)

// refreshConfigLock re-seals the config integrity lock after a CLI command
// legitimately writes active.yaml. Without this, the next command's
// integrity.IsModified check sees the mtime change and falsely warns that
// active.yaml was "modified outside the CLI" — even though this command is
// the CLI. Best-effort: a failure here only means the next run may show a
// stale-lock warning, not a functional break.
func refreshConfigLock(root, activePath string) {
	lockPath := filepath.Join(root, ".config.lock")
	if err := integrity.WriteLock(activePath, lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "[Strategist] WARN: could not refresh config lock: %v\n", err)
	}
}
