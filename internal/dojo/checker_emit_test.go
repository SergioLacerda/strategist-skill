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

func TestCheckEmitLog_KeyPresent(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nranger_done\napproval_prompt\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{
			MustContain:    []string{"ranger_start", "approval_prompt"},
			MustNotContain: []string{"sniper_start"},
		},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckEmitLog_KeyMissing(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"approval_prompt"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckEmitLog_MustNotContainViolated(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nsniper_start\n"), 0o644))

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustNotContain: []string{"sniper_start"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckEmitLog_LogMissing_FilesOnly(t *testing.T) {
	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), true)
	assert.Empty(t, items)
}

func TestCheckEmitLog_LogMissing_NotFilesOnly(t *testing.T) {
	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckEmitLog_NoEmitCriteria(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckEmitLog(criteria, filepath.Join(t.TempDir(), "emit.log"), false)
	assert.Empty(t, items)
}

func TestCheckEmitLog_ReadError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "emit.log")
	require.NoError(t, os.MkdirAll(logPath, 0o755)) // directory, not file — causes read error

	criteria := domain.DojoCriteria{
		EmitLog: domain.DojoEmitLog{MustContain: []string{"ranger_start"}},
	}
	items := dojo.CheckEmitLog(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}
