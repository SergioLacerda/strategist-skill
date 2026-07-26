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

func TestCheckTiming_Pass(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath,
		[]byte("ranger_start\ntotal_wall_time_ms=1200\napproval_prompt\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.True(t, items[0].Passed, "expected pass: %s", items[0].Detail)
}

func TestCheckTiming_Fail_ExceedsMax(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath,
		[]byte("total_wall_time_ms=45000\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "45000")
}

func TestCheckTiming_NilCriteria_Skip(t *testing.T) {
	criteria := domain.DojoCriteria{}
	items := dojo.CheckTiming(criteria, t.TempDir()+"/emit.log", false)
	assert.Empty(t, items)
}

func TestCheckTiming_LogMissing(t *testing.T) {
	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckTiming_FieldMissing(t *testing.T) {
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "emit.log")
	require.NoError(t, os.WriteFile(logPath, []byte("ranger_start\nranger_done\n"), 0o644))

	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "total_wall_time_ms not found")
}

func TestCheckTiming_FilesOnly_Skip(t *testing.T) {
	criteria := domain.DojoCriteria{
		TimingCriteria: &domain.DojoTimingCriteria{MaxWallTimeMs: 30000},
	}
	items := dojo.CheckTiming(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), true)
	assert.Empty(t, items)
}
