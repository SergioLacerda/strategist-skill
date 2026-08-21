package plugins_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	digestA1 = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestA2 = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestB1 = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestC1 = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func TestResolveDeterministicallyPinsHighestCompatibleGraph(t *testing.T) {
	t.Parallel()

	candidates := []plugins.Candidate{
		{ID: "adapter/b", Kind: "adapter_contract", Version: "1.0.0", Digest: digestB1},
		{
			ID:      "package/a",
			Kind:    "package",
			Version: "1.0.0",
			Digest:  digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=1 <2", Reason: "host translation"},
			},
		},
		{
			ID:      "package/a",
			Kind:    "package",
			Version: "1.1.0",
			Digest:  digestA2,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=1 <2", Reason: "host translation"},
			},
		},
	}

	first, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, candidates)
	require.NoError(t, err)
	second, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, reverseCandidates(candidates))
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, "strategist-plugin-lock/v1", first.SchemaVersion)
	assert.Equal(t, []domain.PluginLockNode{
		{ID: "adapter/b", Kind: "adapter_contract", Digest: digestB1},
		{ID: "package/a", Kind: "package", Digest: digestA2},
	}, first.Nodes)
	require.NotEmpty(t, first.GraphDigest)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, first.GraphDigest)
	assert.Equal(t, first.GraphDigest, first.ResolutionID)
}

func TestResolveRejectsDependencyConflictsWithExplanation(t *testing.T) {
	t.Parallel()

	_, err := plugins.Resolve([]plugins.Requirement{
		{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=1 <2"},
		{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=2 <3"},
	}, []plugins.Candidate{
		{ID: "adapter/b", Kind: "adapter_contract", Version: "1.5.0", Digest: digestB1},
		{ID: "adapter/b", Kind: "adapter_contract", Version: "2.0.0", Digest: digestC1},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency_conflict")
	assert.Contains(t, err.Error(), "adapter/b")
	assert.Contains(t, err.Error(), ">=1 <2")
	assert.Contains(t, err.Error(), ">=2 <3")
}

func TestResolveRejectsRequiredDependencyCycle(t *testing.T) {
	t.Parallel()

	_, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, []plugins.Candidate{
		{
			ID:      "package/a",
			Kind:    "package",
			Version: "1.0.0",
			Digest:  digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=1 <2"},
			},
		},
		{
			ID:      "adapter/b",
			Kind:    "adapter_contract",
			Version: "1.0.0",
			Digest:  digestB1,
			Dependencies: []plugins.Dependency{
				{ID: "package/a", Kind: "package", Constraint: ">=1 <2"},
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency_cycle")
	assert.Contains(t, err.Error(), "package/a")
	assert.Contains(t, err.Error(), "adapter/b")
}

func TestResolveSkipsMissingOptionalDependency(t *testing.T) {
	t.Parallel()

	lock, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, []plugins.Candidate{
		{
			ID:      "package/a",
			Kind:    "package",
			Version: "1.0.0",
			Digest:  digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/optional", Kind: "adapter_contract", Constraint: ">=1 <2", Optional: true},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []domain.PluginLockNode{{ID: "package/a", Kind: "package", Digest: digestA1}}, lock.Nodes)
}

func TestVerifyLockReplaysOfflineByDigest(t *testing.T) {
	t.Parallel()

	candidates := []plugins.Candidate{
		{ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA1},
	}
	lock, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, candidates)
	require.NoError(t, err)
	require.NoError(t, plugins.VerifyLock(lock, candidates))

	err = plugins.VerifyLock(lock, []plugins.Candidate{
		{ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA2},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock_replay_digest_missing")
}

func TestVerifyLockRejectsGraphDigestMismatch(t *testing.T) {
	t.Parallel()

	lock := domain.PluginLock{
		SchemaVersion: "strategist-plugin-lock/v1",
		ResolutionID:  digestA1,
		GraphDigest:   digestA1,
		Nodes:         []domain.PluginLockNode{{ID: "package/a", Kind: "package", Digest: digestA1}},
	}
	lock.GraphDigest = digestB1

	err := plugins.VerifyLock(lock, []plugins.Candidate{{ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA1}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock_graph_digest_mismatch")
}

func reverseCandidates(in []plugins.Candidate) []plugins.Candidate {
	out := append([]plugins.Candidate(nil), in...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
