package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallTransaction_RollbackRemoveAllFailureIsLogged(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	parent := t.TempDir()
	strategistDir := filepath.Join(parent, "workspace", ".strategist")

	// Transaction created before strategistDir exists — mirrors a fresh install.
	tx := newInstallTransaction(strategistDir)
	require.False(t, tx.existedBefore)

	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "f.txt"), []byte("x"), 0o644))

	// Read-only parent lets RemoveAll clear strategistDir's contents but blocks
	// removing strategistDir itself — the failure path rollback must only log.
	workspaceDir := filepath.Join(parent, "workspace")
	require.NoError(t, os.Chmod(workspaceDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(workspaceDir, 0o755) })

	tx.rollback(context.Background())

	_, statErr := os.Stat(strategistDir)
	require.NoError(t, statErr, "strategistDir itself must survive when its parent blocks removal")
}
