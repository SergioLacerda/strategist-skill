package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionAdapterVocabularyIsClosed(t *testing.T) {
	require.True(t, ExecutionAdapterCurrentHost.Committable())
	require.True(t, ExecutionAdapterCodexChild.IsChild())
	require.True(t, ExecutionAdapterClaudeChild.IsChild())
	require.False(t, ExecutionAdapterCurrentHost.IsChild())
	require.False(t, ExecutionAdapterCurrentHostUnverified.Committable(), "the compatibility reading is never committed")
	require.True(t, ExecutionAdapterCurrentHostUnverified.Known())
	require.False(t, MissionExecutionAdapter("verified_child").Known())
	require.False(t, MissionExecutionAdapter("").Known())
}

func TestChildAdapterForHostRejectsUnknownHosts(t *testing.T) {
	got, err := ChildAdapterForHost("codex")
	require.NoError(t, err)
	require.Equal(t, ExecutionAdapterCodexChild, got)
	got, err = ChildAdapterForHost("claude")
	require.NoError(t, err)
	require.Equal(t, ExecutionAdapterClaudeChild, got)
	_, err = ChildAdapterForHost("current_host_adapter")
	require.Error(t, err)
}

func TestRecordWithoutAdapterReadsAsUnverifiedCurrentHost(t *testing.T) {
	require.Equal(t, ExecutionAdapterCurrentHostUnverified, MissionInvocationRecord{}.EffectiveAdapter())
	require.Equal(t, ExecutionAdapterCodexChild, MissionInvocationRecord{ExecutionAdapter: ExecutionAdapterCodexChild}.EffectiveAdapter())
}
