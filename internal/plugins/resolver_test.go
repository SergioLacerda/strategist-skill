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

func TestVerifyLockRejectsUnsupportedSchema(t *testing.T) {
	t.Parallel()

	lock := domain.PluginLock{SchemaVersion: "strategist-plugin-lock/v0"}
	err := plugins.VerifyLock(lock, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock_schema_unsupported")
}

func TestResolveSharedDependencySatisfiesBothRequirersWithoutReselecting(t *testing.T) {
	t.Parallel()

	// Two top-level packages both depend on the same adapter with compatible
	// (overlapping) constraints — the second resolveRequirement call for
	// "adapter/shared" must hit the already-resolved fast path instead of
	// selecting again.
	candidates := []plugins.Candidate{
		{
			ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/shared", Kind: "adapter_contract", Constraint: ">=1 <2"},
			},
		},
		{
			ID: "package/b", Kind: "package", Version: "1.0.0", Digest: digestC1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/shared", Kind: "adapter_contract", Constraint: ">=1"},
			},
		},
		{ID: "adapter/shared", Kind: "adapter_contract", Version: "1.0.0", Digest: digestB1},
	}

	lock, err := plugins.Resolve([]plugins.Requirement{
		{ID: "package/a", Kind: "package", Constraint: ">=1 <2"},
		{ID: "package/b", Kind: "package", Constraint: ">=1 <2"},
	}, candidates)
	require.NoError(t, err)
	assert.Len(t, lock.Nodes, 3)
}

func TestResolveRejectsMissingRequiredDependency(t *testing.T) {
	t.Parallel()

	_, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, []plugins.Candidate{
		{
			ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/missing", Kind: "adapter_contract", Constraint: ">=1 <2"},
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency_missing")
	assert.Contains(t, err.Error(), "adapter/missing")
}

func TestResolveBreaksTiesByDigestWhenVersionsAreEqual(t *testing.T) {
	t.Parallel()

	// Same kind/id/version, different digest: compareCandidate must fall
	// through compareVersions (tied) to the digest ordering branches.
	candidates := []plugins.Candidate{
		{ID: "adapter/b", Kind: "adapter_contract", Version: "1.0.0", Digest: digestB1}, // "cccc..."
		{ID: "adapter/b", Kind: "adapter_contract", Version: "1.0.0", Digest: digestC1}, // "dddd..." > digestB1
	}

	lock, err := plugins.Resolve([]plugins.Requirement{{ID: "adapter/b", Kind: "adapter_contract"}}, candidates)
	require.NoError(t, err)
	require.Len(t, lock.Nodes, 1)
	assert.Equal(t, digestB1, lock.Nodes[0].Digest, "lower digest sorts first and wins the tie")

	reversedLock, err := plugins.Resolve([]plugins.Requirement{{ID: "adapter/b", Kind: "adapter_contract"}}, reverseCandidates(candidates))
	require.NoError(t, err)
	assert.Equal(t, lock, reversedLock, "candidate order must not affect the tie-break outcome")
}

func TestResolveOrdersMultipleDependenciesDeterministically(t *testing.T) {
	t.Parallel()

	// A candidate with 2+ dependencies exercises sortedDependencies' actual
	// comparator (a length-1 slice never invokes it).
	candidates := []plugins.Candidate{
		{
			ID: "package/a", Kind: "package", Version: "1.0.0", Digest: digestA1,
			Dependencies: []plugins.Dependency{
				{ID: "adapter/z", Kind: "adapter_contract", Constraint: ">=1"},
				{ID: "adapter/b", Kind: "adapter_contract", Constraint: ">=1"},
			},
		},
		{ID: "adapter/z", Kind: "adapter_contract", Version: "1.0.0", Digest: digestC1},
		{ID: "adapter/b", Kind: "adapter_contract", Version: "1.0.0", Digest: digestB1},
	}

	lock, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1 <2"}}, candidates)
	require.NoError(t, err)
	assert.Len(t, lock.Nodes, 3)
}

func TestResolveConstraintOperatorsAndDefaultsAreHonored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		constraint string
		version    string
		wantErr    bool
	}{
		{"exact match with = prefix", "=1.2.0", "1.2.0", false},
		{"exact mismatch with = prefix", "=1.2.0", "1.3.0", true},
		{"bare version defaults to exact match", "1.2.0", "1.2.0", false},
		{"bare version mismatch", "1.2.0", "1.3.0", true},
		{"empty constraint matches anything", "", "9.9.9", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: tt.constraint}}, []plugins.Candidate{
				{ID: "package/a", Kind: "package", Version: tt.version, Digest: digestA1},
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "dependency_missing")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolveTreatsMalformedVersionSegmentsAsZero(t *testing.T) {
	t.Parallel()

	// versionParts falls back to 0 for both an empty segment ("1..0") and a
	// non-numeric one ("1.x.0") instead of failing resolution outright.
	lock, err := plugins.Resolve([]plugins.Requirement{{ID: "package/a", Kind: "package", Constraint: ">=1.0.0"}}, []plugins.Candidate{
		{ID: "package/a", Kind: "package", Version: "1..0", Digest: digestA1},
		{ID: "package/a", Kind: "package", Version: "1.x.0", Digest: digestA2},
	})
	require.NoError(t, err)
	require.Len(t, lock.Nodes, 1)
}

func reverseCandidates(in []plugins.Candidate) []plugins.Candidate {
	out := append([]plugins.Candidate(nil), in...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
