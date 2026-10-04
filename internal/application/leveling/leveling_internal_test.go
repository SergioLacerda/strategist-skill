package levelingapp

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/require"
)

func TestReuseLevelHandlesHitsMissesAndLedgerErrors(t *testing.T) {
	t.Parallel()

	input := LevelingInput{Mission: "mission-1", Role: "ranger", Run: "run-1"}
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "levels.jsonl")

	result, reused, err := reuseLevel(ledgerPath, input)
	require.NoError(t, err)
	require.False(t, reused)
	require.Empty(t, result)

	result, reused, err = reuseLevel(ledgerPath, LevelingInput{Mission: input.Mission, Role: input.Role, Reason: "explicit"})
	require.NoError(t, err)
	require.False(t, reused)
	require.Empty(t, result)

	require.NoError(t, leveling.AppendRecord(ledgerPath, leveling.Record{
		MissionID: input.Mission,
		Run:       input.Run,
		Level:     leveling.Level{Role: input.Role, Model: "opus", Effort: "high"},
	}))
	result, reused, err = reuseLevel(ledgerPath, input)
	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, "opus", result.Level.Model)
	require.True(t, result.Reused)

	unknownPath := filepath.Join(root, "unknown.jsonl")
	require.NoError(t, leveling.AppendRecord(unknownPath, leveling.Record{
		MissionID: input.Mission,
		Run:       input.Run,
		Level:     leveling.Level{Role: input.Role},
	}))
	_, reused, err = reuseLevel(unknownPath, input)
	require.NoError(t, err)
	require.False(t, reused)

	blocker := filepath.Join(root, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))
	_, _, err = reuseLevel(filepath.Join(blocker, "levels.jsonl"), input)
	require.Error(t, err)
}

func TestResolveLevelHandlesPassthroughSanitizationAndPolicyErrors(t *testing.T) {
	t.Parallel()
	reg := domain.DefaultRoleRegistry()

	result := resolveLevel(reg, func() (leveling.Policy, error) {
		t.Fatal("manual level resolution must not load policy")
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{Mode: domain.LevelingModeManual}, LevelingInput{
		Role:       "Ranger",
		Provider:   "CODEX",
		HostModel:  "<your-model>",
		HostEffort: "not-a-tier",
	})
	require.Equal(t, "ranger", result.Level.Role)
	require.Contains(t, result.Warning, "on_start placeholder")
	require.Contains(t, result.Warning, "not one of")

	result = resolveLevel(reg, func() (leveling.Policy, error) {
		return leveling.Policy{}, errors.New("policy unavailable")
	}, domain.LevelingConfig{}, LevelingInput{Role: "ranger", Provider: "CODEX"})
	require.Contains(t, result.Warning, "policy unavailable")
}

func TestRecordLevelHandlesSkipDuplicateAndAppendBranches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	known := leveling.Level{Role: "ranger", Model: "opus", Effort: "high"}
	unknown := leveling.Level{Role: "ranger"}

	require.NoError(t, recordLevel(filepath.Join(root, "ignored.jsonl"), LevelingInput{}, &LevelingResult{Level: known}, 20))

	gateResult := &LevelingResult{Level: leveling.Level{Role: "gate"}}
	require.NoError(t, recordLevel(filepath.Join(root, "gate.jsonl"), LevelingInput{Mission: "m", Role: "gate"}, gateResult, 20))
	require.False(t, gateResult.Recorded)

	unknownPath := filepath.Join(root, "unknown.jsonl")
	unknownResult := &LevelingResult{Level: unknown}
	require.NoError(t, recordLevel(unknownPath, LevelingInput{Mission: "m", Role: "ranger"}, unknownResult, 20))
	require.True(t, unknownResult.Recorded)

	duplicateResult := &LevelingResult{Level: unknown}
	require.NoError(t, recordLevel(unknownPath, LevelingInput{Mission: "m", Role: "ranger"}, duplicateResult, 20))
	require.False(t, duplicateResult.Recorded)

	knownPath := filepath.Join(root, "known.jsonl")
	require.NoError(t, leveling.AppendRecord(knownPath, leveling.Record{MissionID: "m", Level: known}))
	newResult := &LevelingResult{Level: unknown}
	require.NoError(t, recordLevel(knownPath, LevelingInput{Mission: "m", Role: "ranger"}, newResult, 20))
	require.True(t, newResult.Recorded)

	appendBlocker := filepath.Join(root, "append-blocker")
	require.NoError(t, os.WriteFile(appendBlocker, []byte("not a directory"), 0o600))
	err := recordLevel(filepath.Join(appendBlocker, "levels.jsonl"), LevelingInput{Mission: "m", Role: "ranger", Reason: "manual"}, &LevelingResult{Level: known}, 20)
	require.ErrorContains(t, err, "record role level")

	err = recordLevel(filepath.Join(appendBlocker, "levels.jsonl"), LevelingInput{Mission: "m", Role: "ranger"}, &LevelingResult{Level: unknown}, 20)
	require.ErrorContains(t, err, "read role level ledger")
}

func TestResolveLevelDefaultsMaxRecords(t *testing.T) {
	t.Parallel()

	ledgerPath := filepath.Join(t.TempDir(), "levels.jsonl")
	input := LevelingInput{
		Role:       "ranger",
		Mission:    "mission-1",
		HostModel:  "opus",
		HostEffort: "high",
	}
	result, err := ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, ledgerPath, input, 0)
	require.NoError(t, err)
	require.Equal(t, "ranger", result.Level.Role)
	require.NotEmpty(t, result.Level.Model)

	reused, err := ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		t.Fatal("reused level must not load policy")
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, ledgerPath, input, 20)
	require.NoError(t, err)
	require.True(t, reused.Reused)

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))
	_, err = ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, filepath.Join(blocker, "levels.jsonl"), input, 20)
	require.ErrorContains(t, err, "read role level ledger")
}
