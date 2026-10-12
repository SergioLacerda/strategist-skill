package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
)

func TestValidateRoleSourceParityAcceptsEmbeddedProjection(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, (embed.Extractor{}).Extract(root, false))
	require.Empty(t, validateRoleSourceParity(root))
}

func TestValidateRoleSourceParityReportsMissingAndStaleFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, (embed.Extractor{}).Extract(root, false))
	require.NoError(t, os.Remove(filepath.Join(root, "roles", "ranger.yaml")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "internal_skills", "archivist", "skill.yaml"), []byte("id: archivist\n"), 0o644))

	errs := validateRoleSourceParity(root)
	require.Contains(t, errs, `runtime_missing: Role source file "roles/ranger.yaml" is missing — run strategist install`)
	require.Contains(t, errs, `runtime_stale: Role source file "internal_skills/archivist/skill.yaml" differs from the embedded source — run strategist install`)
}
