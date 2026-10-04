package application_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/require"
)

func TestValidateWorkspaceReportsMissingRuntimeWithoutWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	report := application.ValidateWorkspace(root)
	require.NotEmpty(t, report.Errors)
	require.GreaterOrEqual(t, report.Checks, 1)
	_, err := os.Stat(filepath.Join(root, "active.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
