package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFiles_FileExists(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run")
	require.NoError(t, os.MkdirAll(filepath.Join(runDir, "todo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "todo", "geral.md"),
		[]byte("ideia: KATA_RAPIDO test idea\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{
				Path:             "todo/geral.md",
				RequiredSections: []string{"ideia:"},
				MustContain:      []string{"KATA_RAPIDO"},
			},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	for _, item := range items {
		assert.True(t, item.Passed, "expected pass: %s — %s", item.Label, item.Detail)
	}
}

func TestCheckFiles_FileMissing(t *testing.T) {
	base := t.TempDir()
	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md"},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "not found")
}

func TestCheckFiles_SectionMissing(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("no section here\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", RequiredSections: []string{"ideia:"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	require.Len(t, items, 2)
	assert.True(t, items[0].Passed)
	assert.False(t, items[1].Passed)
}

func TestCheckFiles_MustContainFails(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("ideia: other content\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", MustContain: []string{"KATA_RAPIDO"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
			assert.Contains(t, it.Detail, "KATA_RAPIDO")
		}
	}
	assert.True(t, failed)
}

func TestCheckFiles_MustNotContainFails(t *testing.T) {
	base := t.TempDir()
	runDir := filepath.Join(base, "dojo", "run", "todo")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "geral.md"), []byte("FORBIDDEN_STRING present\n"), 0o644))

	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "todo/geral.md", MustNotContain: []string{"FORBIDDEN_STRING"}},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	failed := false
	for _, it := range items {
		if !it.Passed {
			failed = true
		}
	}
	assert.True(t, failed)
}

func TestCheckFiles_EmptyFilesCreated(t *testing.T) {
	criteria := domain.DojoCriteria{RunDir: "dojo/run"}
	items := dojo.CheckFiles(criteria, t.TempDir())
	assert.Empty(t, items)
}

func TestCheckFiles_PathTraversalRejected(t *testing.T) {
	base := t.TempDir()
	criteria := domain.DojoCriteria{
		RunDir: "dojo/run",
		FilesCreated: []domain.DojoFileCheck{
			{Path: "../../outside.md"},
		},
	}
	items := dojo.CheckFiles(criteria, base)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "escapes root")
	assert.NoFileExists(t, filepath.Join(filepath.Dir(base), "outside.md"))
}
