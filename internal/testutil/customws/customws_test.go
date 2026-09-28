package customws

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceHoldsAnAddedPackageAndADiscoveryBinding(t *testing.T) {
	root := Workspace(t)

	lock, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, err)
	require.Contains(t, string(lock), "installed_instance_id: "+Instance)
	require.Contains(t, string(lock), "installed_instance_id: brainstorming")
	_, statErr := os.Stat(filepath.Join(root, "providers", Instance, "adapter.yaml"))
	require.NoError(t, statErr)
}
