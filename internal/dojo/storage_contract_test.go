package dojo_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStoragePaths_KeepsConfiguredDojoDomain(t *testing.T) {
	base := t.TempDir()

	paths, err := dojo.NewStoragePaths(base, "sample-scenario")

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "dojo"), paths.DojoRoot)
	assert.Equal(t, filepath.Join(base, "dojo", "sample-scenario"), paths.ScenarioDir)
	assert.Equal(t, filepath.Join(base, "dojo", ".last-run", "sample-scenario", "emit.log"), paths.EmitLogPath)
	assert.Equal(t, "dojo/sample-scenario", paths.ScenarioID)
}

func TestNewStoragePaths_RejectsScenarioPathEscape(t *testing.T) {
	for _, scenario := range []string{"../outside", "nested/scenario", `nested\\scenario`, ".", ""} {
		t.Run(scenario, func(t *testing.T) {
			_, err := dojo.NewStoragePaths(t.TempDir(), scenario)
			require.Error(t, err)
		})
	}
}

func TestArtifactPolicies_ClassifyCurrentState(t *testing.T) {
	policies := dojo.ArtifactPolicies()
	require.Len(t, policies, 5)

	byName := make(map[dojo.ArtifactName]dojo.ArtifactPolicy, len(policies))
	for _, policy := range policies {
		byName[policy.Name] = policy
	}
	assert.Equal(t, dojo.DurabilityDurable, byName[dojo.ArtifactCriteria].Durability)
	assert.Equal(t, dojo.DurabilityDurable, byName[dojo.ArtifactLesson].Durability)
	assert.Equal(t, dojo.DurabilityDurable, byName[dojo.ArtifactHistory].Durability)
	assert.Equal(t, dojo.DurabilityRetained, byName[dojo.ArtifactResult].Durability)
	assert.Equal(t, dojo.DurabilityReplaceable, byName[dojo.ArtifactEmitLog].Durability)
	for _, policy := range policies {
		assert.Equal(t, "<base_path>/dojo", policy.Owner)
		assert.NotEmpty(t, policy.BackupExpectation)
		assert.NotEmpty(t, policy.Recovery)
	}
}

func TestPersistResult_ConcurrentRunsKeepHistoryLinesAndValidLatestResult(t *testing.T) {
	base := t.TempDir()
	const runs = 20
	var wg sync.WaitGroup
	errs := make(chan error, runs)

	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := domain.DojoCheckResult{
				Scenario: "concurrent-scenario",
				Items:    []domain.DojoCheckItem{{Label: "pipeline", Passed: true}},
			}
			now := time.Now()
			if err := dojo.PersistResult(base, result, now, now); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	raw, err := os.ReadFile(filepath.Join(base, "dojo", ".last-run", "concurrent-scenario", "result.json"))
	require.NoError(t, err)
	var record dojo.ResultRecord
	require.NoError(t, json.Unmarshal(raw, &record))
	assert.Equal(t, "concurrent-scenario", record.Scenario)

	history, err := os.ReadFile(filepath.Join(base, "dojo", ".history.jsonl"))
	require.NoError(t, err)
	assert.Len(t, splitNonEmptyLines(string(history)), runs)
}

func TestInspectStorage_ReportsCorruptOrphanedState(t *testing.T) {
	base := t.TempDir()
	paths, err := dojo.NewStoragePaths(base, "orphaned-scenario")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(paths.LastRunDir, 0o755))
	require.NoError(t, os.WriteFile(paths.ResultPath, []byte("{"), 0o644))

	report, err := dojo.InspectStorage(base, "orphaned-scenario")

	require.NoError(t, err)
	assert.True(t, report.ResultPresent)
	assert.False(t, report.ResultValid)
	assert.True(t, report.Orphaned)
	assert.True(t, report.RecoveryRequired)
	assert.NotEmpty(t, report.Issues)
}
