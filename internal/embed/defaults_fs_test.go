package embed

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultsFSIsRootedAtDefaults(t *testing.T) {
	t.Parallel()
	fsys := DefaultsFS()
	data, err := fs.ReadFile(fsys, "roles/default.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	_, err = fs.Stat(fsys, "defaults")
	require.Error(t, err, "the defaults/ prefix must be stripped")
}
